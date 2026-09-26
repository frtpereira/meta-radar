package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/frtpereira/meta-radar/internal/config"
	"github.com/frtpereira/meta-radar/internal/db"
	"github.com/frtpereira/meta-radar/internal/ingest"
	"github.com/frtpereira/meta-radar/internal/limitless"
	"github.com/frtpereira/meta-radar/internal/limitlesslabs"
)

func main() {
	game := flag.String("game", "PTCG", "Limitless game id to sync")
	format := flag.String("format", "STANDARD", "Limitless format id to sync (empty = all formats for the game)")
	minPlayers := flag.Int("min-players", 32, "skip tournaments with fewer players than this")
	maxPages := flag.Int("max-pages", 5, "how many pages of /tournaments to walk per pass (50 per page)")
	interval := flag.Duration("interval", 0, "if set (e.g. 15m), run continuously on this interval instead of once")
	requestDelay := flag.Duration("request-delay", 500*time.Millisecond, "pause between tournaments during a sync pass, to stay under the API's rate limit")
	refresh := flag.Duration("refresh", 0, "re-sync tournaments already stored if they were last checked longer ago than this (0 = never re-sync a seen tournament). Use a small value like 1s to force a full re-sync -- e.g. after seeding a new meta, so already-synced tournaments get their archetype_id backfilled.")

	skipOfficial := flag.Bool("skip-official", false, "skip syncing official (offline) Regionals/Worlds tournaments from the labs API")
	officialSampleSize := flag.Int("official-sample-size", 3, "how many of the most recent official events to check per pass")
	officialFormat := flag.String("official-format", "STANDARD", "meta/format code official tournaments attach to")
	officialOrganizer := flag.String("official-organizer", "Play! Pokémon", "organizer_name written for every official tournament")
	officialRequestDelay := flag.Duration("official-request-delay", 500*time.Millisecond, "pause between requests to the (undocumented, unrated-limited) labs API")
	officialRecheck := flag.Duration("official-recheck", 24*time.Hour, "re-sync an official tournament already stored if it still has missing decklists and was last checked longer ago than this (0 = never recheck)")
	officialMaxDecklists := flag.Int("official-max-decklists", 50, "cap on individual decklist fetches per official tournament, per pass (0 = no cap)")
	officialMinDate := flag.String("official-min-date", "2026-08-01", "skip official events that started before this date (YYYY-MM-DD); empty disables the cutoff")
	flag.Parse()

	var minEventDate time.Time
	if *officialMinDate != "" {
		var err error
		minEventDate, err = time.Parse("2006-01-02", *officialMinDate)
		if err != nil {
			log.Fatalf("invalid -official-min-date %q: %v", *officialMinDate, err)
		}
	}

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	client := limitless.NewClient(cfg.LimitlessAPIBase, cfg.LimitlessAPIKey)
	syncer := ingest.NewSyncer(pool, client)
	syncer.LabsClient = limitlesslabs.NewClient(cfg.LabsAPIBase)

	opts := ingest.Options{
		Game:         *game,
		Format:       *format,
		MinPlayers:   *minPlayers,
		MaxPages:     *maxPages,
		RequestDelay: *requestDelay,
		Refresh:      *refresh,
	}

	labsOpts := ingest.LabsOptions{
		SampleSize:         *officialSampleSize,
		FormatCode:         *officialFormat,
		OrganizerName:      *officialOrganizer,
		RequestDelay:       *officialRequestDelay,
		RecheckWindow:      *officialRecheck,
		MaxDecklistFetches: *officialMaxDecklists,
		MinEventDate:       minEventDate,
	}

	runOnce := func() {
		start := time.Now()
		log.Printf("sync starting: game=%s format=%s min_players=%d", opts.Game, opts.Format, opts.MinPlayers)
		if err := syncer.Run(ctx, opts); err != nil {
			log.Printf("sync pass failed: %v", err)
		} else {
			log.Printf("sync finished in %s", time.Since(start))
		}

		if *skipOfficial {
			return
		}
		officialStart := time.Now()
		log.Printf("official tournaments sync starting: sample_size=%d format=%s", labsOpts.SampleSize, labsOpts.FormatCode)
		if err := syncer.RunLabs(ctx, labsOpts); err != nil {
			log.Printf("official tournaments sync pass failed: %v", err)
			return
		}
		log.Printf("official tournaments sync finished in %s", time.Since(officialStart))
	}

	runOnce()

	if *interval > 0 {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for range ticker.C {
			runOnce()
		}
	}
}
