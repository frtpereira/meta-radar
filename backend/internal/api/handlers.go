package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/ingest"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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
// basic input for the "top performing decks" view. avg_standing excludes
// drops (standing = 0) via NULLIF, since a drop isn't a real finishing
// placement and averaging it in would make dropping look like winning.
func (h *Handler) ArchetypeStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metaID := r.URL.Query().Get("meta_id")
	if metaID == "" {
		writeError(w, http.StatusBadRequest, "meta_id is required")
		return
	}

	query := `
		SELECT a.id, a.name, a.slug, COUNT(d.id) AS deck_count,
		       AVG(NULLIF(s.standing, 0)) AS avg_standing,
		       COUNT(*) FILTER (WHERE s.standing = 0) AS drop_count
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
		ID          int64    `json:"id"`
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		DeckCount   int      `json:"deck_count"`
		AvgStanding *float64 `json:"avg_standing"`
		DropCount   int      `json:"drop_count"`
	}

	stats := []archetypeStat{}
	for rows.Next() {
		var s archetypeStat
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.DeckCount, &s.AvgStanding, &s.DropCount); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning archetype stat: "+err.Error())
			return
		}
		stats = append(stats, s)
	}

	writeJSON(w, http.StatusOK, stats)
}

// ArchetypeDetail returns one archetype's metadata plus its computed core
// card list (populated by cmd/cluster; nil until that's been run).
func (h *Handler) ArchetypeDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var (
		a              models.Archetype
		coreCardsJSON  []byte
		coreThreshold  *float64
		coreComputedAt *time.Time
	)
	err := h.DB.QueryRow(ctx, `
		SELECT id, meta_id::text, name, slug, core_cards, core_threshold, core_computed_at
		FROM archetypes WHERE id = $1`, id,
	).Scan(&a.ID, &a.MetaID, &a.Name, &a.Slug, &coreCardsJSON, &coreThreshold, &coreComputedAt)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "archetype not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying archetype: "+err.Error())
		return
	}

	var coreCards []models.Card
	if len(coreCardsJSON) > 0 {
		_ = json.Unmarshal(coreCardsJSON, &coreCards)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":               a.ID,
		"meta_id":          a.MetaID,
		"name":             a.Name,
		"slug":             a.Slug,
		"core_cards":       coreCards,
		"core_threshold":   coreThreshold,
		"core_computed_at": coreComputedAt,
	})
}

// ArchetypeVariants groups an archetype's decklists by core_hash -- each
// group is one distinct build (skeleton), separate from tech-choice noise.
// Requires cmd/cluster to have run for this archetype's meta first;
// decklists with a NULL core_hash (not yet clustered) are excluded.
func (h *Handler) ArchetypeVariants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	query := `
		SELECT d.core_hash, COUNT(*) AS deck_count,
		       AVG(NULLIF(s.standing, 0)) AS avg_standing,
		       COUNT(*) FILTER (WHERE s.standing = 0) AS drop_count,
		       -- one representative decklist per variant, for showing its tech choices
		       (ARRAY_AGG(d.id ORDER BY d.id))[1] AS sample_decklist_id
		FROM decklists d
		LEFT JOIN standings s ON s.decklist_id = d.id
		WHERE d.archetype_id = $1 AND d.core_hash IS NOT NULL
		GROUP BY d.core_hash
		ORDER BY deck_count DESC`

	rows, err := h.DB.Query(ctx, query, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying variants: "+err.Error())
		return
	}
	defer rows.Close()

	type variant struct {
		CoreHash         string   `json:"core_hash"`
		DeckCount        int      `json:"deck_count"`
		AvgStanding      *float64 `json:"avg_standing"`
		DropCount        int      `json:"drop_count"`
		SampleDecklistID int64    `json:"sample_decklist_id"`
	}

	variants := []variant{}
	for rows.Next() {
		var v variant
		if err := rows.Scan(&v.CoreHash, &v.DeckCount, &v.AvgStanding, &v.DropCount, &v.SampleDecklistID); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning variant: "+err.Error())
			return
		}
		variants = append(variants, v)
	}

	writeJSON(w, http.StatusOK, variants)
}

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
