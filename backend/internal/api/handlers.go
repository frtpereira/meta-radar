package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/ingest"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	DB            *pgxpool.Pool
	Syncer        *ingest.Syncer
	WebhookSecret string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListTournaments supports ?min_players=64&format=STANDARD&meta_id=... so the
// frontend can drive the "64+ player, current meta" filter directly via
// query params rather than filtering client-side.
func (h *Handler) ListTournaments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	minPlayers := 0
	if v := q.Get("min_players"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			minPlayers = parsed
		}
	}
	format := q.Get("format")
	metaID := q.Get("meta_id")

	query := `
		SELECT id, name, game, format_code, meta_id, date, players, is_online, has_decklists, organizer_name
		FROM tournaments
		WHERE players >= $1
		  AND ($2 = '' OR format_code = $2)
		  AND ($3 = '' OR meta_id::text = $3)
		ORDER BY date DESC
		LIMIT 200`

	rows, err := h.DB.Query(ctx, query, minPlayers, format, metaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying tournaments: "+err.Error())
		return
	}
	defer rows.Close()

	tournaments := []models.Tournament{}
	for rows.Next() {
		var t models.Tournament
		if err := rows.Scan(&t.ID, &t.Name, &t.Game, &t.FormatCode, &t.MetaID, &t.Date, &t.Players, &t.IsOnline, &t.HasDecklists, &t.OrganizerName); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning tournament: "+err.Error())
			return
		}
		tournaments = append(tournaments, t)
	}

	writeJSON(w, http.StatusOK, tournaments)
}

func (h *Handler) ListMetas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.DB.Query(ctx, `SELECT id, name, format_code, starts_at, ends_at FROM metas ORDER BY starts_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying metas: "+err.Error())
		return
	}
	defer rows.Close()

	metas := []models.Meta{}
	for rows.Next() {
		var m models.Meta
		if err := rows.Scan(&m.ID, &m.Name, &m.FormatCode, &m.StartsAt, &m.EndsAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning meta: "+err.Error())
			return
		}
		metas = append(metas, m)
	}

	writeJSON(w, http.StatusOK, metas)
}

// ArchetypeStats returns per-archetype play counts for a given meta, the
// basic input for the "top performing decks" view. This starts simple
// (count + avg placing); win-rate needs the standings join added once
// pairings data is being ingested.
func (h *Handler) ArchetypeStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metaID := r.URL.Query().Get("meta_id")
	if metaID == "" {
		writeError(w, http.StatusBadRequest, "meta_id is required")
		return
	}

	query := `
		SELECT a.id, a.name, a.slug, COUNT(d.id) AS deck_count, AVG(s.standing) AS avg_placing
		FROM archetypes a
		JOIN decklists d ON d.archetype_id = a.id
		LEFT JOIN standings s ON s.decklist_id = d.id
		WHERE a.meta_id = $1
		GROUP BY a.id, a.name, a.slug
		ORDER BY deck_count DESC`

	rows, err := h.DB.Query(ctx, query, metaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying archetype stats: "+err.Error())
		return
	}
	defer rows.Close()

	type archetypeStat struct {
		ID         int64   `json:"id"`
		Name       string  `json:"name"`
		Slug       string  `json:"slug"`
		DeckCount  int     `json:"deck_count"`
		AvgPlacing float64 `json:"avg_placing"`
	}

	stats := []archetypeStat{}
	for rows.Next() {
		var s archetypeStat
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.DeckCount, &s.AvgPlacing); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning archetype stat: "+err.Error())
			return
		}
		stats = append(stats, s)
	}

	writeJSON(w, http.StatusOK, stats)
}

// LimitlessWebhook receives the tournament:ended notification and triggers
// a sync for that single tournament. Limitless calls this synchronously and
// doesn't document a retry policy, so we acknowledge immediately (202) and
// run the actual sync in the background rather than making Limitless wait
// on our full fetch-standings-and-upsert round trip.
func (h *Handler) LimitlessWebhook(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Secret string `json:"secret"`
		Event  struct {
			Name         string `json:"name"`
			TournamentID string `json:"tournamentId"`
			Game         string `json:"game"`
		} `json:"event"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if h.WebhookSecret != "" && payload.Secret != h.WebhookSecret {
		writeError(w, http.StatusUnauthorized, "invalid secret")
		return
	}

	if payload.Event.TournamentID == "" {
		writeError(w, http.StatusBadRequest, "missing event.tournamentId")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "received"})

	go func() {
		// Detached from the request context on purpose -- it must outlive
		// the HTTP handler, which has already responded above.
		ctx := context.Background()
		if err := h.Syncer.SyncOne(ctx, payload.Event.TournamentID); err != nil {
			log.Printf("webhook-triggered sync failed for %s: %v", payload.Event.TournamentID, err)
		}
	}()
}
