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
	"time"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/limitless"
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

// DefaultOptions matches the brief: 64+ player events, standard format.
func DefaultOptions() Options {
	return Options{
		Game:         "PTCG",
		Format:       "STANDARD",
		MinPlayers:   64,
		MaxPages:     5,
		Refresh:      0,
		RequestDelay: 500 * time.Millisecond,
	}
}

// Run walks recent tournaments, skips ones that don't meet the player
// threshold, and syncs (or re-syncs, per Refresh) the rest.
func (s *Syncer) Run(ctx context.Context, opts Options) error {
	for page := 1; page <= max(opts.MaxPages, 1); page++ {
		summaries, err := s.Client.ListTournaments(ctx, opts.Game, opts.Format, 50, page)
		if err != nil {
			return fmt.Errorf("listing tournaments (page %d): %w", page, err)
		}
		if len(summaries) == 0 {
			break // no more pages
		}

		for _, t := range summaries {
			if t.Players < opts.MinPlayers {
				continue
			}

			shouldSync, err := s.shouldSync(ctx, t.ID, opts.Refresh)
			if err != nil {
				log.Printf("checking sync state for %s: %v", t.ID, err)
				continue
			}
			if !shouldSync {
				continue
			}

			if err := s.syncTournament(ctx, t.ID); err != nil {
				log.Printf("syncing tournament %s (%s): %v", t.ID, t.Name, err)
				s.logSync(ctx, t.ID, "poll", "error", err.Error())
				continue
			}
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
