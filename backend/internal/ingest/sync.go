// Package ingest pulls tournament, standings and decklist data from the
// Limitless API and upserts it into Postgres. It's meant to be driven two
// ways: a periodic poll (cmd/ingest running on an interval) and, later, the
// webhook handler in internal/api enqueuing a sync for a single tournament
// id as soon as it ends.
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/frtpereira/meta-radar/internal/limitless"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Syncer struct {
	DB     *pgxpool.Pool
	Client *limitless.Client
}

func NewSyncer(db *pgxpool.Pool, client *limitless.Client) *Syncer {
	return &Syncer{DB: db, Client: client}
}

// Options controls one sync pass.
type Options struct {
	Game         string        // e.g. "PTCG"
	Format       string        // e.g. "STANDARD"; empty = all formats for the game
	MinPlayers   int           // tournaments below this are skipped entirely
	MaxPages     int           // safety cap on how many pages of /tournaments to walk
	Refresh      time.Duration // re-sync tournaments already seen within this window? 0 = never re-sync a seen tournament
	RequestDelay time.Duration // pause between tournaments, to stay under the API's rate limit
}

// DefaultOptions matches the brief: 32+ player events, standard format.
func DefaultOptions() Options {
	return Options{
		Game:         "PTCG",
		Format:       "STANDARD",
		MinPlayers:   32,
		MaxPages:     5,
		Refresh:      0,
		RequestDelay: 500 * time.Millisecond,
	}
}

// Run walks recent tournaments, skips ones that don't meet the player
// threshold, and syncs (or re-syncs, per Refresh) the rest.
func (s *Syncer) Run(ctx context.Context, opts Options) error {
	var seen, synced, skipped, failed int

	for page := 1; page <= max(opts.MaxPages, 1); page++ {
		log.Printf("fetching page %d of tournaments...", page)
		summaries, err := s.Client.ListTournaments(ctx, opts.Game, opts.Format, 50, page)
		if err != nil {
			return fmt.Errorf("listing tournaments (page %d): %w", page, err)
		}
		if len(summaries) == 0 {
			break // no more pages
		}
		log.Printf("page %d: %d tournaments", page, len(summaries))

		for _, t := range summaries {
			seen++
			if t.Players < opts.MinPlayers {
				skipped++
				continue
			}

			shouldSync, err := s.shouldSync(ctx, t.ID, opts.Refresh)
			if err != nil {
				log.Printf("checking sync state for %s: %v", t.ID, err)
				continue
			}
			if !shouldSync {
				skipped++
				continue
			}

			log.Printf("syncing %s (%s, %d players)...", t.ID, t.Name, t.Players)
			if err := s.syncTournament(ctx, t.ID); err != nil {
				failed++
				log.Printf("  failed: %v", err)
				s.logSync(ctx, t.ID, "poll", "error", err.Error())
				continue
			}
			synced++
			log.Printf("  synced (%d/%d checked so far, %d skipped, %d failed)", synced, seen, skipped, failed)
			s.logSync(ctx, t.ID, "poll", "success", "")

			if opts.RequestDelay > 0 {
				select {
				case <-time.After(opts.RequestDelay):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	log.Printf("pass complete: %d checked, %d synced, %d skipped, %d failed", seen, synced, skipped, failed)

	// Refresh materialized view for matchups after an ingestion pass so the
	// query-backed endpoint can serve pre-aggregated data.
	if _, err := s.DB.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`); err != nil {
		// REFRESH CONCURRENTLY requires a unique index; if it fails, log and continue.
		log.Printf("failed to refresh matchups_mv: %v", err)
	}

	return nil
}

// SyncOne is the entrypoint the webhook handler will call for a single
// tournament id once it's wired up.
func (s *Syncer) SyncOne(ctx context.Context, tournamentID string) error {
	if err := s.syncTournament(ctx, tournamentID); err != nil {
		s.logSync(ctx, tournamentID, "webhook", "error", err.Error())
		return err
	}
	s.logSync(ctx, tournamentID, "webhook", "success", "")

	// Refresh the materialized view so webhook-driven syncs make data
	// available to clients promptly. Ignore refresh errors but log them.
	if _, err := s.DB.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`); err != nil {
		log.Printf("failed to refresh matchups_mv after webhook sync %s: %v", tournamentID, err)
	}

	return nil
}

func (s *Syncer) shouldSync(ctx context.Context, tournamentID string, refresh time.Duration) (bool, error) {
	var lastChecked time.Time
	err := s.DB.QueryRow(ctx, `SELECT last_checked_at FROM tournaments WHERE id = $1`, tournamentID).Scan(&lastChecked)
	if err == pgx.ErrNoRows {
		return true, nil // never seen -- sync it
	}
	if err != nil {
		return false, err
	}
	if refresh <= 0 {
		return false, nil // already have it, and we don't re-sync completed tournaments by default
	}
	return time.Since(lastChecked) >= refresh, nil
}

func (s *Syncer) syncTournament(ctx context.Context, tournamentID string) error {
	details, err := s.Client.GetTournamentDetails(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("fetching details: %w", err)
	}

	standings, err := s.Client.GetStandings(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("fetching standings: %w", err)
	}

	pairings, err := s.Client.GetPairings(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("fetching pairings: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	_, err = tx.Exec(ctx, `
		INSERT INTO tournaments (id, name, game, format_code, date, players, is_online, is_public, has_decklists, organizer_name, raw_details, synced_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now(), now())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			players = EXCLUDED.players,
			is_online = EXCLUDED.is_online,
			is_public = EXCLUDED.is_public,
			has_decklists = EXCLUDED.has_decklists,
			organizer_name = EXCLUDED.organizer_name,
			raw_details = EXCLUDED.raw_details,
			last_checked_at = now()`,
		details.ID, details.Name, details.Game, details.Format, details.Date, details.Players,
		details.IsOnline, details.IsPublic, details.Decklists, details.Organizer.Name, detailsJSON(details),
	)
	if err != nil {
		return fmt.Errorf("upserting tournament: %w", err)
	}

	// Try to attach this tournament to an existing open meta for its format.
	// (Meta creation/rotation is a deliberate, human decision -- see README --
	// so we only attach, never create one here.)
	var metaID *string
	_ = tx.QueryRow(ctx, `SELECT id::text FROM metas WHERE format_code = $1 AND ends_at IS NULL`, details.Format).Scan(&metaID)
	if metaID != nil {
		if _, err := tx.Exec(ctx, `UPDATE tournaments SET meta_id = $1 WHERE id = $2`, *metaID, details.ID); err != nil {
			return fmt.Errorf("attaching meta: %w", err)
		}
	}

	for _, entry := range standings {
		if err := s.upsertStandingEntry(ctx, tx, details.ID, metaID, entry); err != nil {
			return fmt.Errorf("upserting standing for player %s: %w", entry.Player, err)
		}
	}

	// Runs after the standings loop above on purpose: every player who
	// appears in this tournament's standings (Limitless includes drops --
	// standing 0 -- so that's effectively the full roster) already has
	// their real display name in `players` by this point. replacePairings
	// only falls back to using a raw player id as the name for someone who
	// appears in pairings but was *never* in standings at all -- plausible
	// for a disqualification or a bye-placeholder id, but not for an
	// ordinary drop, since those are already covered above.
	if err := s.replacePairings(ctx, tx, details.ID, pairings); err != nil {
		return fmt.Errorf("upserting pairings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (s *Syncer) upsertStandingEntry(ctx context.Context, tx pgx.Tx, tournamentID string, metaID *string, entry limitless.StandingEntry) error {
	if entry.Player == "" {
		return nil // shouldn't happen, but don't let a malformed row kill the whole sync
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO players (id, name) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`,
		entry.Player, entry.Name,
	)
	if err != nil {
		return fmt.Errorf("upserting player: %w", err)
	}

	var archetypeID *int64
	if entry.Deck != nil && metaID != nil {
		id, err := s.upsertArchetype(ctx, tx, *metaID, entry.Deck.ID, entry.Deck.Name)
		if err != nil {
			return fmt.Errorf("upserting archetype: %w", err)
		}
		archetypeID = &id
	}

	var decklistID *int64
	if len(entry.Decklist) > 0 && string(entry.Decklist) != "null" {
		cards := limitless.ParsePTCGDecklist(entry.Decklist)
		id, err := s.upsertDecklist(ctx, tx, tournamentID, entry.Player, archetypeID, cards, entry.Decklist)
		if err != nil {
			return fmt.Errorf("upserting decklist: %w", err)
		}
		decklistID = &id
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO standings (tournament_id, player_id, standing, wins, losses, ties, decklist_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tournament_id, player_id) DO UPDATE SET
			standing = EXCLUDED.standing,
			wins = EXCLUDED.wins,
			losses = EXCLUDED.losses,
			ties = EXCLUDED.ties,
			decklist_id = EXCLUDED.decklist_id`,
		tournamentID, entry.Player, entry.Placing, entry.Record.Wins, entry.Record.Losses, entry.Record.Ties, decklistID,
	)
	if err != nil {
		return fmt.Errorf("upserting standing row: %w", err)
	}

	return nil
}

func (s *Syncer) upsertArchetype(ctx context.Context, tx pgx.Tx, metaID, slug, name string) (int64, error) {
	if slug == "" {
		slug = "uncategorized"
	}
	if name == "" {
		name = "Uncategorized"
	}

	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO archetypes (meta_id, name, slug)
		VALUES ($1, $2, $3)
		ON CONFLICT (meta_id, slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`,
		metaID, name, slug,
	).Scan(&id)
	return id, err
}

func (s *Syncer) upsertDecklist(ctx context.Context, tx pgx.Tx, tournamentID, playerID string, archetypeID *int64, cards any, raw any) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO decklists (tournament_id, player_id, archetype_id, cards, raw_list)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tournament_id, player_id) DO UPDATE SET
			archetype_id = EXCLUDED.archetype_id,
			cards = EXCLUDED.cards,
			raw_list = EXCLUDED.raw_list
		RETURNING id`,
		tournamentID, playerID, archetypeID, cards, raw,
	).Scan(&id)
	return id, err
}

// replacePairings replaces a tournament's pairings wholesale (delete then
// reinsert) rather than upserting -- there's no natural per-row identity to
// conflict on that's cheaper than just recomputing, and a tournament's
// pairings never partially change in practice (they're either not final
// yet, in which case we won't have synced it, or they are).
func (s *Syncer) replacePairings(ctx context.Context, tx pgx.Tx, tournamentID string, pairings []limitless.PairingEntry) error {
	if _, err := tx.Exec(ctx, `DELETE FROM pairings WHERE tournament_id = $1`, tournamentID); err != nil {
		return fmt.Errorf("clearing previous pairings: %w", err)
	}

	for _, p := range pairings {
		if p.Player1 == "" && p.Player2 == "" {
			continue
		}

		// Only a fallback: if this player was already seen in standings
		// (the normal case -- see the comment at the call site), this
		// ON CONFLICT DO NOTHING leaves their real name untouched.
		if p.Player1 != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO players (id, name) VALUES ($1, $2)
				ON CONFLICT (id) DO NOTHING`,
				p.Player1, p.Player1,
			); err != nil {
				return fmt.Errorf("ensuring player1 exists: %w", err)
			}
		}
		if p.Player2 != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO players (id, name) VALUES ($1, $2)
				ON CONFLICT (id) DO NOTHING`,
				p.Player2, p.Player2,
			); err != nil {
				return fmt.Errorf("ensuring player2 exists: %w", err)
			}
		}

		winnerPlayerID, recognized := normalizeWinnerPlayerID(p.Winner, p.Player1, p.Player2)
		result := classifyPairingResult(winnerPlayerID, recognized, p.Player1, p.Player2)

		if !recognized && p.Player2 != "" {
			// The raw `winner` value was non-empty and not a recognized
			// "no winner" sentinel, but didn't match either player id --
			// that's a sign of a parsing assumption being wrong (id format
			// mismatch, unexpected API shape) rather than a genuine draw.
			// Stored as "unknown" and excluded from win/draw-based stats
			// (see ArchetypeStats/MatchupStats, which filter on
			// result IN ('win','draw')) instead of silently counting as a
			// tie, so bad data can't quietly inflate tie rates.
			log.Printf("pairing tournament=%s phase=%d round=%d table=%d: winner value %q didn't match player1=%q or player2=%q -- storing as unknown",
				tournamentID, p.Phase, p.Round, p.Table, string(p.Winner), p.Player1, p.Player2)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO pairings (tournament_id, phase, round, table_number, player1_id, player2_id, winner_player_id, result, raw_pairing)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8, $9)`,
			tournamentID, p.Phase, p.Round, p.Table, p.Player1, p.Player2, winnerPlayerID, result, pairingJSON(p),
		); err != nil {
			return fmt.Errorf("inserting pairing phase=%d round=%d table=%d: %w", p.Phase, p.Round, p.Table, err)
		}
	}

	return nil
}

// normalizeWinnerPlayerID parses the raw `winner` field from a pairing.
//
// Returns ("", true) for a confirmed no-winner case (empty, JSON null, or
// the -1 sentinel observed elsewhere in this API for non-decisive results)
// -- that's a real draw. Returns (id, true) when the value is a JSON string
// matching player1 or player2 -- a confirmed win. Returns ("", false) for
// anything else: a non-empty value that doesn't fit either recognized
// shape. That last case is NOT a draw -- it's ambiguous/unparseable data,
// and the caller (classifyPairingResult) must not treat it as one.
func normalizeWinnerPlayerID(raw json.RawMessage, player1, player2 string) (winnerID string, recognized bool) {
	v := strings.TrimSpace(string(raw))
	if v == "" || v == "null" || v == "-1" {
		return "", true
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == player1 || s == player2 {
			return s, true
		}
		return "", false // present, valid JSON string, but matches neither player
	}

	return "", false // not a shape we recognize at all
}

// classifyPairingResult turns a normalized winner id into a result label.
// "unknown" (rather than "draw") is used whenever the winner value
// couldn't be confidently classified, so downstream win/draw-based stats
// (which filter on result IN ('win','draw')) don't silently absorb
// unparseable data as if it were a real tie.
func classifyPairingResult(winnerPlayerID string, recognized bool, player1, player2 string) string {
	if player2 == "" {
		if winnerPlayerID != "" {
			return "bye"
		}
		return "unknown"
	}
	if !recognized {
		return "unknown"
	}
	if winnerPlayerID == player1 || winnerPlayerID == player2 {
		return "win"
	}
	return "draw"
}

func pairingJSON(p limitless.PairingEntry) []byte {
	b, err := json.Marshal(p)
	if err != nil {
		return nil
	}
	return b
}

func (s *Syncer) logSync(ctx context.Context, tournamentID, source, status, detail string) {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO sync_log (tournament_id, source, status, detail)
		VALUES ($1, $2, $3, $4)`,
		tournamentID, source, status, detail,
	)
	if err != nil {
		log.Printf("failed to write sync_log for %s: %v", tournamentID, err)
	}
}

func detailsJSON(d *limitless.TournamentDetails) []byte {
	b, err := json.Marshal(d)
	if err != nil {
		return nil
	}
	return b
}
