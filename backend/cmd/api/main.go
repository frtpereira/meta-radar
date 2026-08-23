package main

import (
	"context"
	"log"
	"net/http"

	"github.com/frtpereira/meta-radar/internal/api"
	"github.com/frtpereira/meta-radar/internal/config"
	"github.com/frtpereira/meta-radar/internal/db"
	"github.com/frtpereira/meta-radar/internal/ingest"
	"github.com/frtpereira/meta-radar/internal/limitless"
)

// @title Meta Radar API
// @version 1.0
// @description API for tracking Pokemon TCG tournaments, decklists, archetypes and matchup statistics.
// @BasePath /
func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	client := limitless.NewClient(cfg.LimitlessAPIBase, cfg.LimitlessAPIKey)
	syncer := ingest.NewSyncer(pool, client)

	router := api.NewRouter(pool, syncer, cfg.WebhookSecret)

	if cfg.TLSEnabled() {
		log.Printf("listening on :%s (HTTPS)", cfg.Port)
		if err := http.ListenAndServeTLS(":"+cfg.Port, cfg.TLSCertFile, cfg.TLSKeyFile, router); err != nil {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	log.Printf("listening on :%s (HTTP)", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
