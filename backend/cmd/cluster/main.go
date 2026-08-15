package main

import (
	"context"
	"flag"
	"log"

	"github.com/frtpereira/meta-radar/internal/archetype"
	"github.com/frtpereira/meta-radar/internal/config"
	"github.com/frtpereira/meta-radar/internal/db"
)

func main() {
	metaID := flag.String("meta", "", "meta id to cluster (empty = every meta in the database)")
	threshold := flag.Float64("threshold", archetype.DefaultCoreThreshold, "fraction of an archetype's decklists a card must appear in to count as core (0-1)")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	cl := archetype.NewClusterer(pool)

	if *metaID != "" {
		log.Printf("clustering meta %s (threshold %.2f)...", *metaID, *threshold)
		if err := cl.RunForMeta(ctx, *metaID, *threshold); err != nil {
			log.Fatalf("clustering meta %s: %v", *metaID, err)
		}
		log.Println("done")
		return
	}

	rows, err := pool.Query(ctx, `SELECT id FROM metas`)
	if err != nil {
		log.Fatalf("listing metas: %v", err)
	}
	var metaIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("scanning meta id: %v", err)
		}
		metaIDs = append(metaIDs, id)
	}
	rows.Close()

	for _, id := range metaIDs {
		log.Printf("clustering meta %s (threshold %.2f)...", id, *threshold)
		if err := cl.RunForMeta(ctx, id, *threshold); err != nil {
			log.Printf("  failed: %v", err)
			continue
		}
	}
	log.Println("done")
}
