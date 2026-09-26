// labs_sync.go adds official (offline) Regionals/Worlds ingestion, from
// limitlesstcg.com's labs API (internal/limitlesslabs), alongside the
// existing Play API ingestion in sync.go. See db/migrations/0013's comment
// for the schema additions this relies on (ingest_source, division,
// event_id).
package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/frtpereira/meta-radar/internal/limitless"
	"github.com/frtpereira/meta-radar/internal/limitlesslabs"
	"github.com/jackc/pgx/v5"
)

// LabsOptions controls one official-tournaments sync pass.
type LabsOptions struct {
	// SampleSize is how many of the most recent official events to check
	// each pass. Official tournaments are far less frequent than online
	// ones, so this only needs to be a handful -- new events get picked
	// up well before they'd fall off the sample.
	SampleSize int
	// FormatCode is the meta/format official tournaments attach to (see
	// resolveMeta in sync.go). Official Regionals/Worlds are assumed to
	// always be Standard; change this if that's ever not the case.
	FormatCode string
	// OrganizerName is written as every official tournament's
	// organizer_name, so tournament_organizer alone identifies "official"
	// tournaments elsewhere (filters, frontend display) without a
	// separate flag.
	OrganizerName string
	// RequestDelay pauses between HTTP calls, to stay polite to an
	// undocumented API that was never built to be called directly.
	RequestDelay time.Duration
	// RecheckWindow: an already-synced official event gets re-synced if
	// it still has standings rows with no decklist (see
	// staleLabsEventIDs) AND it was last checked longer ago than this.
	// 0 disables rechecking.
	RecheckWindow time.Duration
	// MaxDecklistFetches caps how many individual per-player decklist
	// calls happen in one pass, per division. A Worlds-sized field can
	// have ~800 missing decklists on the very first sync; fetching all
	// of them serially (with RequestDelay between each) in one pass could
	// take a very long time, so the rest are picked up by later passes
	// via the recheck loop. 0 = no cap.
	MaxDecklistFetches int
	// MinEventDate excludes official events that started before this
	// date -- per project scope, only tournaments from PBL onwards
	// (August 2026) are ingested. An event's date comes from
	// TournamentMeta.Date (parsed via limitlesslabs.ParseEventDate); a
	// tournament that predates this is skipped, not treated as a
	// failure. Zero value disables the cutoff.
	MinEventDate time.Time
}

// DefaultLabsOptions matches the brief discussed for this feature: a small
// sample per pass, Standard format, and a fixed organizer name.
//
// OrganizerName is a placeholder pending confirmation of the exact string
// to use -- easy to change here, in one place.
func DefaultLabsOptions() LabsOptions {
	return LabsOptions{
		SampleSize:         3,
		FormatCode:         "STANDARD",
		OrganizerName:      "Play! Pokémon",
		RequestDelay:       500 * time.Millisecond,
		RecheckWindow:      24 * time.Hour,
		MaxDecklistFetches: 50,
		MinEventDate:       time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
	}
}

// errLabsEventBeforeCutoff is returned by syncLabsDivision (never wrapped
// with additional context) when an event's date is before
// LabsOptions.MinEventDate, so RunLabs can tell a deliberate skip apart
// from an actual sync failure.
var errLabsEventBeforeCutoff = errors.New("official event predates the ingestion cutoff")

// RunLabs syncs the most recent official tournaments, plus any
// already-synced official tournament still missing decklists. Each
// official event is ingested as up to len(limitlesslabs.Divisions)
// separate tournament rows (see labsTournamentID) sharing one event_id.
func (s *Syncer) RunLabs(ctx context.Context, opts LabsOptions) error {
	if s.LabsClient == nil {
		return fmt.Errorf("RunLabs called without a LabsClient configured")
	}
	if opts.SampleSize <= 0 {
		opts.SampleSize = 3
	}
	if opts.FormatCode == "" {
		opts.FormatCode = "STANDARD"
	}
	if opts.OrganizerName == "" {
		opts.OrganizerName = "Play! Pokémon"
	}

	entries, err := s.LabsClient.ListTournaments(ctx)
	if err != nil {
		return fmt.Errorf("listing official tournaments: %w", err)
	}

	recentIDs, skippedUnfinished := selectRecentEventIDs(entries, opts.SampleSize)
	if skippedUnfinished > 0 {
		log.Printf("official tournaments: ignoring %d not-yet-completed event(s) from the tail of the list", skippedUnfinished)
	}

	stale, err := s.staleLabsEventIDs(ctx, opts.RecheckWindow)
	if err != nil {
		log.Printf("checking for official tournaments with missing decklists: %v", err)
	}

	seen := make(map[string]bool, len(recentIDs)+len(stale))
	var eventIDs []string
	for _, id := range recentIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		eventIDs = append(eventIDs, id)
	}
	for _, id := range stale {
		if seen[id] {
			continue
		}
		seen[id] = true
		eventIDs = append(eventIDs, id)
	}

	var synced, skipped, failed int
	for _, eventID := range eventIDs {
		for _, division := range limitlesslabs.Divisions {
			tournamentID := labsTournamentID(eventID, division)
			log.Printf("syncing official tournament %s...", tournamentID)
			if err := s.syncLabsDivision(ctx, eventID, division, opts); err != nil {
				if errors.Is(err, errLabsEventBeforeCutoff) {
					skipped++
					log.Printf("  skipping: predates %s cutoff", opts.MinEventDate.Format("2006-01-02"))
					s.logSync(ctx, tournamentID, "labs-poll", "skipped", err.Error())
					continue
				}
				failed++
				log.Printf("  failed: %v", err)
				s.logSync(ctx, tournamentID, "labs-poll", "error", err.Error())
				continue
			}
			synced++
			s.logSync(ctx, tournamentID, "labs-poll", "success", "")

			if opts.RequestDelay > 0 {
				select {
				case <-time.After(opts.RequestDelay):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	log.Printf("official tournaments pass complete: %d checked, %d synced, %d skipped (before cutoff), %d failed", synced+skipped+failed, synced, skipped, failed)

	if _, err := s.DB.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`); err != nil {
		log.Printf("failed to refresh matchups_mv: %v", err)
	}
	return nil
}

// staleLabsEventIDs returns the bare event ids (not the "labs-" prefixed
// tournament id or event_id) of official events that still have a
// standing with no decklist (excluding standing = 0, i.e. dropped/DQ'd
// players -- see upsertLabsStandingEntry) and haven't been checked
// recently. A whole event (all divisions) is re-synced even if only one
// division is actually incomplete, to keep this simple; re-fetching an
// already-complete division is cheap since decklist fetches are skipped
// per-player once stored (see backfillLabsDecklists).
func (s *Syncer) staleLabsEventIDs(ctx context.Context, window time.Duration) ([]string, error) {
	if window <= 0 {
		return nil, nil
	}
	cutoff := time.Now().Add(-window)

	rows, err := s.DB.Query(ctx, `
		SELECT DISTINCT t.event_id FROM tournaments t
		WHERE t.ingest_source = 'labs'
		  AND t.last_checked_at < $1
		  AND EXISTS (
			SELECT 1 FROM standings s
			WHERE s.tournament_id = t.id AND s.decklist_id IS NULL AND s.standing != 0
		  )`,
		cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var eventID string
		if err := rows.Scan(&eventID); err != nil {
			return nil, err
		}
		ids = append(ids, strings.TrimPrefix(eventID, "labs-"))
	}
	return ids, rows.Err()
}

func (s *Syncer) syncLabsDivision(ctx context.Context, eventID, division string, opts LabsOptions) error {
	meta, err := s.LabsClient.GetTournamentMeta(ctx, eventID, division)
	if err != nil {
		return fmt.Errorf("fetching tournament meta: %w", err)
	}

	date, err := limitlesslabs.ParseEventDate(meta.Date)
	if err != nil {
		return fmt.Errorf("parsing event date %q: %w", meta.Date, err)
	}
	if !opts.MinEventDate.IsZero() && date.Before(opts.MinEventDate) {
		return errLabsEventBeforeCutoff
	}

	standings, err := s.LabsClient.GetStandings(ctx, eventID, division)
	if err != nil {
		return fmt.Errorf("fetching standings: %w", err)
	}

	var rounds []labsRoundPairings
	for round := 1; round <= meta.Round; round++ {
		entries, err := s.LabsClient.GetPairings(ctx, eventID, division, round)
		if err != nil {
			return fmt.Errorf("fetching pairings round %d: %w", round, err)
		}
		rounds = append(rounds, labsRoundPairings{round: round, entries: entries})

		if opts.RequestDelay > 0 {
			select {
			case <-time.After(opts.RequestDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	tournamentID := labsTournamentID(eventID, division)
	name := labsEventName(meta, division)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	_, err = tx.Exec(ctx, `
		INSERT INTO tournaments (id, name, game, format_code, date, players, is_online, is_public, has_decklists, organizer_name, ingest_source, division, event_id, raw_details, synced_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, true, $7, $8, 'labs', $9, $10, $11, now(), now())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			players = EXCLUDED.players,
			has_decklists = EXCLUDED.has_decklists,
			raw_details = EXCLUDED.raw_details,
			last_checked_at = now()`,
		tournamentID, name, "PTCG", opts.FormatCode, date, meta.Players,
		meta.Decklists != 0, opts.OrganizerName, division, labsEventGroupID(eventID), labsJSON(meta),
	)
	if err != nil {
		return fmt.Errorf("upserting tournament: %w", err)
	}

	metaID, err := s.resolveMeta(ctx, tx, opts.FormatCode)
	if err != nil {
		return err
	}
	if metaID != nil {
		if _, err := tx.Exec(ctx, `UPDATE tournaments SET meta_id = $1, is_current_standard = true WHERE id = $2`, *metaID, tournamentID); err != nil {
			return fmt.Errorf("attaching meta: %w", err)
		}
	}

	for _, entry := range standings {
		if err := s.upsertLabsStandingEntry(ctx, tx, tournamentID, entry); err != nil {
			return fmt.Errorf("upserting standing for player %d: %w", entry.TPID, err)
		}
	}

	// Every player who appears in pairings is expected to already exist
	// (from the standings loop just above): unlike the Play API, pairing
	// player ids here are the *same* per-tournament sequential id as
	// StandingEntry.TPID (see PairingEntry's doc comment), which is
	// exactly the set standings just enumerated -- so there's no
	// "player only ever seen in pairings" fallback case to handle here.
	if err := s.replaceLabsPairings(ctx, tx, tournamentID, rounds); err != nil {
		return fmt.Errorf("upserting pairings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	// Decklists are fetched one HTTP call per player, so they run after
	// the main transaction commits rather than holding it open across
	// potentially hundreds of sequential requests.
	s.backfillLabsDecklists(ctx, tournamentID, eventID, metaID, standings, opts)

	return nil
}

func (s *Syncer) upsertLabsStandingEntry(ctx context.Context, tx pgx.Tx, tournamentID string, entry limitlesslabs.StandingEntry) error {
	// Keyed on TPID, not PlayerID: TPID is the per-tournament sequential id
	// that PairingEntry.Player1/Player2/Winner actually reference (see
	// StandingEntry.TPID's doc comment) -- using PlayerID here would leave
	// pairings referencing player rows that were never inserted.
	playerID := labsPlayerID(entry.TPID)

	if _, err := tx.Exec(ctx, `
		INSERT INTO players (id, name) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`,
		playerID, entry.Name,
	); err != nil {
		return fmt.Errorf("upserting player: %w", err)
	}

	// Matches the existing "0 = dropped" convention used for Play API
	// tournaments (see limitless.StandingEntry.Placing's doc comment):
	// labs tracks drops and DQs as separate flags rather than a placement
	// of 0, so fold both into the same sentinel here.
	standing := entry.Placement
	if entry.Dropped != 0 || entry.DQed != 0 {
		standing = 0
	}

	// decklist_id is deliberately left out of the UPDATE SET below: it's
	// only ever set by backfillLabsDecklists once a decklist is actually
	// fetched, and this upsert must not clobber that back to NULL on a
	// later re-sync pass.
	if _, err := tx.Exec(ctx, `
		INSERT INTO standings (tournament_id, player_id, standing, wins, losses, ties)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tournament_id, player_id) DO UPDATE SET
			standing = EXCLUDED.standing,
			wins = EXCLUDED.wins,
			losses = EXCLUDED.losses,
			ties = EXCLUDED.ties`,
		tournamentID, playerID, standing, entry.Wins, entry.Losses, entry.Ties,
	); err != nil {
		return fmt.Errorf("upserting standing row: %w", err)
	}

	return nil
}

type labsRoundPairings struct {
	round   int
	entries []limitlesslabs.PairingEntry
}

// replaceLabsPairings mirrors replacePairings' delete-then-reinsert
// approach (see that function's doc comment for why), across every round
// fetched for this division.
func (s *Syncer) replaceLabsPairings(ctx context.Context, tx pgx.Tx, tournamentID string, rounds []labsRoundPairings) error {
	if _, err := tx.Exec(ctx, `DELETE FROM pairings WHERE tournament_id = $1`, tournamentID); err != nil {
		return fmt.Errorf("clearing previous pairings: %w", err)
	}

	// Official tournaments' labs data has no phase concept (no separate
	// "day 2"/"top cut" phase number the way the Play API models phases);
	// everything is recorded as phase 1.
	const phase = 1

	for _, rp := range rounds {
		for _, p := range rp.entries {
			if p.Player1 == 0 && p.Player2 == 0 {
				continue
			}

			var player1ID, player2ID *string
			if p.Player1 != 0 {
				id := labsPlayerID(p.Player1)
				player1ID = &id
			}
			if p.Player2 != 0 {
				id := labsPlayerID(p.Player2)
				player2ID = &id
			}

			result := "unknown"
			var winnerID *string
			switch {
			case p.Player2 == 0: // bye
				if p.Winner != 0 {
					result = "bye"
					winnerID = player1ID
				} else {
					result = "unknown"
				}
			case p.Winner == 0:
				result = "draw"
			case p.Winner == p.Player1:
				result = "win"
				winnerID = player1ID
			case p.Winner == p.Player2:
				result = "win"
				winnerID = player2ID
			default:
				// Same defensive stance as normalizeWinnerPlayerID in
				// sync.go: a winner that matches neither player is a sign
				// something about this assumption is wrong, not a draw.
				log.Printf("pairing tournament=%s round=%d table=%d: winner %d didn't match player1=%d or player2=%d -- storing as unknown",
					tournamentID, rp.round, p.Table, p.Winner, p.Player1, p.Player2)
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO pairings (tournament_id, phase, round, table_number, player1_id, player2_id, winner_player_id, result, raw_pairing)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				tournamentID, phase, rp.round, p.Table, player1ID, player2ID, winnerID, result, labsJSON(p),
			); err != nil {
				return fmt.Errorf("inserting pairing round=%d table=%d: %w", rp.round, p.Table, err)
			}
		}
	}

	return nil
}

// backfillLabsDecklists fetches decklists for players labs says have one
// (entry.Decklist == 1) but that we don't already have stored, up to
// opts.MaxDecklistFetches. Errors fetching an individual decklist are
// logged and skipped -- one bad/missing player shouldn't fail the whole
// division's sync, and it'll be retried on a later pass via
// staleLabsEventIDs.
func (s *Syncer) backfillLabsDecklists(ctx context.Context, tournamentID, eventID string, metaID *string, standings []limitlesslabs.StandingEntry, opts LabsOptions) {
	have := map[string]bool{}
	rows, err := s.DB.Query(ctx, `SELECT player_id FROM standings WHERE tournament_id = $1 AND decklist_id IS NOT NULL`, tournamentID)
	if err != nil {
		log.Printf("checking existing decklists for %s: %v", tournamentID, err)
	} else {
		for rows.Next() {
			var playerID string
			if err := rows.Scan(&playerID); err == nil {
				have[playerID] = true
			}
		}
		rows.Close()
	}

	fetched := 0
	for _, entry := range standings {
		if opts.MaxDecklistFetches > 0 && fetched >= opts.MaxDecklistFetches {
			break
		}
		if entry.Decklist == 0 {
			continue
		}
		playerID := labsPlayerID(entry.TPID)
		if have[playerID] {
			continue
		}

		if err := s.fetchAndStoreLabsDecklist(ctx, tournamentID, eventID, metaID, entry.TPID, entry.DeckID, entry.DeckName); err != nil {
			log.Printf("  decklist for player %d in %s: %v", entry.TPID, tournamentID, err)
			continue
		}
		fetched++

		if opts.RequestDelay > 0 {
			select {
			case <-time.After(opts.RequestDelay):
			case <-ctx.Done():
				return
			}
		}
	}

	if fetched > 0 {
		log.Printf("  fetched %d new decklist(s) for %s", fetched, tournamentID)
	}
}

func (s *Syncer) fetchAndStoreLabsDecklist(ctx context.Context, tournamentID, eventID string, metaID *string, tpID int, deckID, deckName string) error {
	decklist, err := s.LabsClient.GetDecklist(ctx, eventID, tpID)
	if err != nil {
		return fmt.Errorf("fetching decklist: %w", err)
	}
	if decklist == nil {
		return fmt.Errorf("labs API returned no decklist for this player")
	}

	// DecklistEntry's shape matches ParsePTCGDecklist's expected input
	// exactly (pokemon/trainer/energy, each {count,name,set,number}) --
	// round-trip through JSON to reuse that parser rather than
	// duplicating it.
	raw, err := json.Marshal(decklist)
	if err != nil {
		return fmt.Errorf("marshaling decklist: %w", err)
	}
	cards := limitless.ParsePTCGDecklist(raw)
	if len(cards) == 0 {
		return fmt.Errorf("decklist response didn't match the expected pokemon/trainer/energy shape")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var archetypeID *int64
	if deckID != "" && metaID != nil {
		id, err := s.upsertArchetype(ctx, tx, *metaID, deckID, deckName)
		if err != nil {
			return fmt.Errorf("upserting archetype: %w", err)
		}
		archetypeID = &id
	}

	playerIDStr := labsPlayerID(tpID)
	decklistID, err := s.upsertDecklist(ctx, tx, tournamentID, playerIDStr, archetypeID, cards, raw)
	if err != nil {
		return fmt.Errorf("upserting decklist: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE standings SET decklist_id = $1 WHERE tournament_id = $2 AND player_id = $3`,
		decklistID, tournamentID, playerIDStr,
	); err != nil {
		return fmt.Errorf("linking decklist to standing: %w", err)
	}

	return tx.Commit(ctx)
}

// labsTournamentID, labsPlayerID and labsEventGroupID are all prefixed
// with "labs-" so labs-sourced ids can never collide with Play API ids in
// the shared tournaments/players tables, regardless of what either
// upstream's own id format happens to look like.
func labsTournamentID(eventID, division string) string {
	return fmt.Sprintf("labs-%s-%s", eventID, division)
}

// selectRecentEventIDs returns the ids of the sampleSize most recent
// *completed* events from a GET /tournaments response, plus how many
// trailing entries were skipped for not being completed yet (the list
// includes scheduled-but-not-yet-run events -- see TournamentListEntry's
// doc comment). Entries are assumed chronological with the most recent
// last, per product knowledge; completed ones are then taken from the
// tail so an upcoming event never displaces a real recent one from the
// sample.
func selectRecentEventIDs(entries []limitlesslabs.TournamentListEntry, sampleSize int) (ids []string, skippedUnfinished int) {
	if sampleSize <= 0 {
		sampleSize = 3
	}
	for i := len(entries) - 1; i >= 0 && len(ids) < sampleSize; i-- {
		e := entries[i]
		if e.ID == "" {
			continue
		}
		if !e.Completed {
			skippedUnfinished++
			continue
		}
		ids = append(ids, e.ID)
	}
	// ids was built newest-first by walking backwards; reverse it so
	// callers see oldest-of-the-sample first, matching the previous
	// slice-based behavior and keeping log output chronological.
	for l, r := 0, len(ids)-1; l < r; l, r = l+1, r-1 {
		ids[l], ids[r] = ids[r], ids[l]
	}
	return ids, skippedUnfinished
}

func labsPlayerID(playerID int) string {
	return fmt.Sprintf("labs-%d", playerID)
}

func labsEventGroupID(eventID string) string {
	return "labs-" + eventID
}

func labsEventName(meta *limitlesslabs.TournamentMeta, division string) string {
	base := labsEventTypeLabel(meta.Type)
	if meta.City != "" {
		base += " " + meta.City
	} else if meta.Name != nil && *meta.Name != "" {
		base = *meta.Name
	}
	return base + " (" + labsDivisionLabel(division) + ")"
}

// labsEventTypeLabel turns the labs API's lowercase `type` (e.g. "worlds",
// confirmed live; "regional" assumed by analogy but not yet observed) into
// the display form used in a tournament name, e.g. "Regional San Diego" --
// matching the existing convention where a tournament's kind is implicit
// in its name rather than a separate field.
func labsEventTypeLabel(t string) string {
	switch t {
	case "":
		return "Official Tournament"
	case "worlds":
		return "Worlds"
	case "regional":
		return "Regional"
	case "international":
		return "International Championship"
	case "special":
		return "Special Event"
	default:
		return strings.ToUpper(t[:1]) + t[1:]
	}
}

func labsDivisionLabel(division string) string {
	switch division {
	case "MA":
		return "Masters"
	case "SR":
		return "Seniors"
	case "JR":
		return "Juniors"
	default:
		return division
	}
}

func labsJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
