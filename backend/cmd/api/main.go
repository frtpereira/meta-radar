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

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
