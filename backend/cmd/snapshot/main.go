// Command snapshot computes and stores a point-in-time archetype
// breakdown (deck share, win rate, avg standing) for one or every
// currently-open meta, on a daily or weekly cadence -- see
// db/migrations/0011_meta_snapshots.sql. It's the write side of the
// meta-snapshot foundation; nothing reads meta_snapshots /
// meta_snapshot_archetypes yet, but this is what later winrate/usage-
// over-time graphs will read from.
//
// Two ways to run it:
//   - One-shot (for `make snapshot-daily` / `make snapshot-weekly`, or
//     manual backfills): pass -type=daily|weekly, optionally -meta=<id>
//     to snapshot just one meta. Runs once and exits.
//   - Daemon (the snapshot-scheduler service in docker-compose.yml):
//     pass -daemon, optionally -at=HH:MM (UTC, default 06:00). Sleeps
//     until the next occurrence of that time every day and takes a
//     full daily snapshot (every currently-open meta), plus a weekly
//     snapshot on top every Monday -- see snapshotDate's comment on why
//     which day of the week the weekly one runs on doesn't matter for
//     correctness, only for which date it's filed under.
//
// Re-running for a date that already has a snapshot replaces its
// archetype rows rather than accumulating duplicates, so the daemon
// re-triggering the same day (e.g. after a restart) is harmless.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/frtpereira/meta-radar/internal/config"
	"github.com/frtpereira/meta-radar/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// snapshotArchetypeQuery computes, for a set of set-meta ids, the
// per-archetype numbers a meta_snapshot_archetypes row needs. Passing
// more than one id (a standard meta's children -- see scopeMetaIDs)
// merges archetypes sharing a slug across those metas into one row,
// the same way internal/api's archetypeStatsStandardQuery merges them
// for /api/archetypes/stats: deck and match totals are aggregated
// independently by slug before joining, so the decklist join can't fan
// out and inflate a summed match count. Passing a single id behaves
// the same as merging within that one meta (a no-op, since slugs are
// already unique per meta), so one query serves both cases.
const snapshotArchetypeQuery = `
	WITH sides AS (
		SELECT d.archetype_id, p.player1_id AS player_id, p.winner_player_id
		FROM pairings p
		JOIN tournaments t ON t.id = p.tournament_id
		JOIN decklists d ON d.tournament_id = p.tournament_id AND d.player_id = p.player1_id
			WHERE t.meta_id = ANY($1::uuid[]) AND p.result IN ('win', 'draw')

		UNION ALL

		SELECT d.archetype_id, p.player2_id AS player_id, p.winner_player_id
		FROM pairings p
		JOIN tournaments t ON t.id = p.tournament_id
		JOIN decklists d ON d.tournament_id = p.tournament_id AND d.player_id = p.player2_id
			WHERE t.meta_id = ANY($1::uuid[]) AND p.result IN ('win', 'draw')
	), match_stats AS (
		SELECT archetype_id,
		       COUNT(*)::int AS matches,
		       SUM(CASE WHEN winner_player_id = player_id THEN 1 ELSE 0 END)::int AS wins,
		       SUM(CASE WHEN winner_player_id IS NOT NULL AND winner_player_id <> player_id THEN 1 ELSE 0 END)::int AS losses,
		       SUM(CASE WHEN winner_player_id IS NULL THEN 1 ELSE 0 END)::int AS ties
		FROM sides
		GROUP BY archetype_id
	), slugs AS (
		SELECT slug, (ARRAY_AGG(id ORDER BY id DESC))[1] AS representative_id
		FROM archetypes
		WHERE meta_id = ANY($1::uuid[]) AND name <> 'Other'
		GROUP BY slug
	), deck_stats AS (
		SELECT a.slug,
		       COUNT(d.id) AS deck_count,
		       AVG(NULLIF(s.standing, 0)) AS avg_standing
		FROM archetypes a
		JOIN decklists d ON d.archetype_id = a.id
		LEFT JOIN standings s ON s.decklist_id = d.id
		WHERE a.meta_id = ANY($1::uuid[]) AND a.name <> 'Other'
		GROUP BY a.slug
	), match_totals AS (
		SELECT a.slug,
		       COALESCE(SUM(ms.matches), 0)::int AS matches,
		       COALESCE(SUM(ms.wins), 0)::int AS wins,
		       COALESCE(SUM(ms.losses), 0)::int AS losses,
		       COALESCE(SUM(ms.ties), 0)::int AS ties
		FROM archetypes a
		LEFT JOIN match_stats ms ON ms.archetype_id = a.id
		WHERE a.meta_id = ANY($1::uuid[]) AND a.name <> 'Other'
		GROUP BY a.slug
	)
	SELECT sl.representative_id, ds.deck_count, ds.avg_standing,
	       mt.matches, mt.wins, mt.losses, mt.ties,
	       CASE WHEN mt.wins + mt.losses = 0 THEN NULL
	            ELSE mt.wins::float8 / (mt.wins + mt.losses)::float8 END AS win_rate
	FROM slugs sl
	JOIN deck_stats ds ON ds.slug = sl.slug
	JOIN match_totals mt ON mt.slug = sl.slug`

type targetMeta struct {
	id       string
	metaType string
}

type archetypeRow struct {
	archetypeID int64
	deckCount   int
	avgStanding *float64
	matches     int
	wins        int
	losses      int
	ties        int
	winRate     *float64
}

func main() {
	snapshotType := flag.String("type", "", `snapshot cadence for a one-shot run: "daily" or "weekly" (ignored with -daemon)`)
	only := flag.String("meta", "", "meta id to snapshot for a one-shot run (empty = every currently open meta, standard and set; ignored with -daemon, which always does every meta)")
	daemon := flag.Bool("daemon", false, "run continuously: sleep until -at (UTC) each day, take a daily snapshot, and also a weekly one every Monday, instead of running once and exiting")
	at := flag.String("at", "06:00", `UTC time of day the daemon runs at each day, as "HH:MM" (only used with -daemon)`)
	flag.Parse()

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if *daemon {
		hour, minute, err := parseHourMinute(*at)
		if err != nil {
			log.Fatalf("invalid -at %q: %v", *at, err)
		}
		runDaemon(ctx, pool, hour, minute)
		return
	}

	if *snapshotType != "daily" && *snapshotType != "weekly" {
		log.Fatalf(`-type must be "daily" or "weekly", got %q`, *snapshotType)
	}
	runPass(ctx, pool, *snapshotType, *only)
}

// parseHourMinute parses an "HH:MM" string (as validated by -at) into
// its hour/minute components.
func parseHourMinute(s string) (hour, minute int, err error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

// runDaemon sleeps until the next occurrence of hour:minute UTC, then
// takes a daily snapshot (and, on Mondays, a weekly one too), forever,
// until ctx is cancelled (SIGINT/SIGTERM).
func runDaemon(ctx context.Context, pool *pgxpool.Pool, hour, minute int) {
	log.Printf("snapshot daemon started: will run daily at %02d:%02d UTC (plus weekly every Monday)", hour, minute)

	for {
		now := time.Now().UTC()
		next := nextOccurrence(now, hour, minute)

		log.Printf("snapshot daemon: sleeping %s, next run at %s", next.Sub(now).Round(time.Second), next.Format(time.RFC3339))

		select {
		case <-ctx.Done():
			log.Println("snapshot daemon: shutting down")
			return
		case <-time.After(next.Sub(now)):
		}

		runPass(ctx, pool, "daily", "")
		if next.Weekday() == time.Monday {
			runPass(ctx, pool, "weekly", "")
		}
	}
}

// nextOccurrence returns the next time hour:minute UTC occurs at or
// after now -- today if that time hasn't passed yet, otherwise
// tomorrow.
func nextOccurrence(now time.Time, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// runPass snapshots `only` (or every currently-open meta, if empty) for
// the given cadence, logging each meta's outcome. Used by both the
// one-shot and daemon code paths.
func runPass(ctx context.Context, pool *pgxpool.Pool, snapshotType, only string) {
	dateStr := snapshotDate(snapshotType).Format("2006-01-02")

	targets, err := targetMetas(ctx, pool, only)
	if err != nil {
		log.Printf("resolving target metas for %s snapshot: %v", snapshotType, err)
		return
	}
	if len(targets) == 0 {
		log.Printf("no open metas to snapshot for %s %s", snapshotType, dateStr)
		return
	}

	for _, target := range targets {
		if err := snapshotOne(ctx, pool, target, snapshotType, dateStr); err != nil {
			log.Printf("snapshotting meta %s (%s): %v", target.id, target.metaType, err)
			continue
		}
		log.Printf("snapshotted meta %s (%s) for %s %s", target.id, target.metaType, snapshotType, dateStr)
	}
}

// snapshotDate returns the date a snapshot taken right now represents.
// For "daily" that's just today (UTC). For "weekly" it's the Monday of
// the current ISO week, so a week's data always lands on the same row
// regardless of which day of the week the job actually runs on.
func snapshotDate(snapshotType string) time.Time {
	now := time.Now().UTC()
	if snapshotType != "weekly" {
		return now
	}
	daysSinceMonday := (int(now.Weekday()) + 6) % 7 // Weekday(): Sunday=0..Saturday=6
	return now.AddDate(0, 0, -daysSinceMonday)
}

// targetMetas resolves which metas to snapshot: just `only` if given,
// otherwise every currently open meta (standard and set both -- see
// db/migrations/0009_meta_hierarchy.sql).
func targetMetas(ctx context.Context, pool *pgxpool.Pool, only string) ([]targetMeta, error) {
	query := `SELECT id::text, meta_type FROM metas WHERE ends_at IS NULL`
	args := []any{}
	if only != "" {
		query = `SELECT id::text, meta_type FROM metas WHERE id = $1::uuid`
		args = []any{only}
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []targetMeta
	for rows.Next() {
		var m targetMeta
		if err := rows.Scan(&m.id, &m.metaType); err != nil {
			return nil, err
		}
		targets = append(targets, m)
	}
	return targets, rows.Err()
}

// scopeMetaIDs returns the set-meta ids a snapshot of target should be
// computed over: itself, unless it's a standard meta, in which case
// every set meta currently nested under it -- mirrors what
// internal/api's resolveMetaScope does for the read path.
func scopeMetaIDs(ctx context.Context, pool *pgxpool.Pool, target targetMeta) ([]string, error) {
	if target.metaType != "standard" {
		return []string{target.id}, nil
	}

	rows, err := pool.Query(ctx, `SELECT id::text FROM metas WHERE parent_meta_id = $1::uuid`, target.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func snapshotOne(ctx context.Context, pool *pgxpool.Pool, target targetMeta, snapshotType, dateStr string) error {
	scope, err := scopeMetaIDs(ctx, pool, target)
	if err != nil {
		return fmt.Errorf("resolving scope: %w", err)
	}
	if len(scope) == 0 {
		// A standard meta with no set meta under it yet -- nothing to
		// snapshot (shouldn't normally happen once `make seed-meta` has
		// run, but a fresh/mid-rotation format can hit this briefly).
		return nil
	}

	rows, err := pool.Query(ctx, snapshotArchetypeQuery, scope)
	if err != nil {
		return fmt.Errorf("querying archetype stats: %w", err)
	}

	var archetypeRows []archetypeRow
	totalDecks := 0
	for rows.Next() {
		var a archetypeRow
		if err := rows.Scan(&a.archetypeID, &a.deckCount, &a.avgStanding, &a.matches, &a.wins, &a.losses, &a.ties, &a.winRate); err != nil {
			rows.Close()
			return fmt.Errorf("scanning archetype row: %w", err)
		}
		archetypeRows = append(archetypeRows, a)
		totalDecks += a.deckCount
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating archetype rows: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	var snapshotID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO meta_snapshots (meta_id, snapshot_type, snapshot_date, total_decks)
		VALUES ($1::uuid, $2, $3::date, $4)
		ON CONFLICT (meta_id, snapshot_type, snapshot_date) DO UPDATE SET
			generated_at = now(),
			total_decks = EXCLUDED.total_decks
		RETURNING id`,
		target.id, snapshotType, dateStr, totalDecks,
	).Scan(&snapshotID)
	if err != nil {
		return fmt.Errorf("upserting snapshot header: %w", err)
	}

	// Re-running for a date that already has a snapshot replaces its
	// archetype rows rather than accumulating duplicates.
	if _, err := tx.Exec(ctx, `DELETE FROM meta_snapshot_archetypes WHERE snapshot_id = $1`, snapshotID); err != nil {
		return fmt.Errorf("clearing previous archetype rows: %w", err)
	}

	for _, a := range archetypeRows {
		var sharePct *float64
		if totalDecks > 0 {
			v := float64(a.deckCount) / float64(totalDecks)
			sharePct = &v
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO meta_snapshot_archetypes
				(snapshot_id, archetype_id, deck_count, share_pct, matches, wins, losses, ties, win_rate, avg_standing)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			snapshotID, a.archetypeID, a.deckCount, sharePct, a.matches, a.wins, a.losses, a.ties, a.winRate, a.avgStanding,
		); err != nil {
			return fmt.Errorf("inserting archetype row for archetype %d: %w", a.archetypeID, err)
		}
	}

	return tx.Commit(ctx)
}
