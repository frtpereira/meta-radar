package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/config"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/db"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/ingest"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/limitless"
)

func main() {
	game := flag.String("game", "PTCG", "Limitless game id to sync")
	format := flag.String("format", "STANDARD", "Limitless format id to sync (empty = all formats for the game)")
	minPlayers := flag.Int("min-players", 64, "skip tournaments with fewer players than this")
	maxPages := flag.Int("max-pages", 5, "how many pages of /tournaments to walk per pass (50 per page)")
	interval := flag.Duration("interval", 0, "if set (e.g. 15m), run continuously on this interval instead of once")
	requestDelay := flag.Duration("request-delay", 500*time.Millisecond, "pause between tournaments during a sync pass, to stay under the API's rate limit")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	client := limitless.NewClient(cfg.LimitlessAPIBase, cfg.LimitlessAPIKey)
	syncer := ingest.NewSyncer(pool, client)

	opts := ingest.Options{
		Game:         *game,
		Format:       *format,
		MinPlayers:   *minPlayers,
		MaxPages:     *maxPages,
		RequestDelay: *requestDelay,
	}

	runOnce := func() {
		start := time.Now()
		log.Printf("sync starting: game=%s format=%s min_players=%d", opts.Game, opts.Format, opts.MinPlayers)
		if err := syncer.Run(ctx, opts); err != nil {
			log.Printf("sync pass failed: %v", err)
			return
		}
		log.Printf("sync finished in %s", time.Since(start))
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