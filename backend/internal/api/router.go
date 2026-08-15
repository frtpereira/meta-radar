package api

import (
	"net/http"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/ingest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func NewRouter(pool *pgxpool.Pool, syncer *ingest.Syncer, webhookSecret string, redisClient ...*redis.Client) http.Handler {
	var rc *redis.Client
	if len(redisClient) > 0 {
		rc = redisClient[0]
	}
	h := &Handler{DB: pool, Syncer: syncer, WebhookSecret: webhookSecret, Redis: rc}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// Tighten this to your actual frontend origin before deploying.
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
	}))

	r.Get("/health", h.Health)

	r.Route("/api", func(r chi.Router) {
		r.Get("/tournaments", h.ListTournaments)
		r.Get("/tournaments/{id}", h.TournamentDetail)
		r.Get("/metas", h.ListMetas)
		r.Get("/archetypes/stats", h.ArchetypeStats)
		r.Get("/matchups/stats", h.MatchupStats)
		r.Get("/archetypes/{id}", h.ArchetypeDetail)
		r.Get("/archetypes/{id}/variants", h.ArchetypeVariants)
		r.Post("/webhooks/limitless", h.LimitlessWebhook)
	})

	return r
}
