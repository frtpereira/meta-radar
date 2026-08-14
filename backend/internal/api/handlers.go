package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/ingest"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v11"
	"os"
)

type Handler struct {
	DB            *pgxpool.Pool
	Syncer        *ingest.Syncer
	WebhookSecret string
	Redis         *redis.Client
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
//
// win_rate is computed from actual pairings (see the `pairings` table),
// not derived from standing/placement -- it only requires the archetype's
// own side of a pairing to be known (the opponent's decklist doesn't need
// to be public), which is a looser requirement than /api/matchups/stats
// needs for a specific matchup. It's null until cmd/ingest has synced
// pairings for this meta's tournaments (needs `make migrate` + a resync
// for anything synced before pairings existed) -- see README.
func (h *Handler) ArchetypeStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metaID := r.URL.Query().Get("meta_id")
	if metaID == "" {
		writeError(w, http.StatusBadRequest, "meta_id is required")
		return
	}

	query := `
		WITH sides AS (
			SELECT d.archetype_id, p.player1_id AS player_id, p.winner_player_id
			FROM pairings p
			JOIN tournaments t ON t.id = p.tournament_id
			JOIN decklists d ON d.tournament_id = p.tournament_id AND d.player_id = p.player1_id
				WHERE t.meta_id = $1::uuid AND p.result IN ('win', 'draw')

			UNION ALL

			SELECT d.archetype_id, p.player2_id AS player_id, p.winner_player_id
			FROM pairings p
			JOIN tournaments t ON t.id = p.tournament_id
			JOIN decklists d ON d.tournament_id = p.tournament_id AND d.player_id = p.player2_id
				WHERE t.meta_id = $1::uuid AND p.result IN ('win', 'draw')
		), match_stats AS (
			SELECT archetype_id,
			       COUNT(*)::int AS matches,
			       SUM(CASE WHEN winner_player_id = player_id THEN 1 ELSE 0 END)::int AS wins,
			       SUM(CASE WHEN winner_player_id IS NOT NULL AND winner_player_id <> player_id THEN 1 ELSE 0 END)::int AS losses,
			       SUM(CASE WHEN winner_player_id IS NULL THEN 1 ELSE 0 END)::int AS ties
			FROM sides
			GROUP BY archetype_id
		)
		SELECT a.id, a.name, a.slug, COUNT(d.id) AS deck_count,
		       AVG(NULLIF(s.standing, 0)) AS avg_standing,
		       COUNT(*) FILTER (WHERE s.standing = 0) AS drop_count,
		       COALESCE(ms.matches, 0), COALESCE(ms.wins, 0), COALESCE(ms.losses, 0), COALESCE(ms.ties, 0),
		       CASE WHEN COALESCE(ms.matches, 0) = 0 THEN NULL
		            ELSE (COALESCE(ms.wins, 0) + 0.5 * COALESCE(ms.ties, 0)) / COALESCE(ms.matches, 0)::float8 END AS score_rate,
		       CASE WHEN COALESCE(ms.wins, 0) + COALESCE(ms.losses, 0) = 0 THEN NULL
		            ELSE COALESCE(ms.wins, 0)::float8 / (COALESCE(ms.wins, 0) + COALESCE(ms.losses, 0))::float8 END AS win_rate
		FROM archetypes a
		JOIN decklists d ON d.archetype_id = a.id
		LEFT JOIN standings s ON s.decklist_id = d.id
		LEFT JOIN match_stats ms ON ms.archetype_id = a.id
				WHERE a.meta_id = $1::uuid
		GROUP BY a.id, a.name, a.slug, ms.matches, ms.wins, ms.losses, ms.ties
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
		Matches     int      `json:"matches"`
		Wins        int      `json:"wins"`
		Losses      int      `json:"losses"`
		Ties        int      `json:"ties"`
		ScoreRate   *float64 `json:"score_rate"`
		WinRate     *float64 `json:"win_rate"`
	}

	stats := []archetypeStat{}
	for rows.Next() {
		var s archetypeStat
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.DeckCount, &s.AvgStanding, &s.DropCount,
			&s.Matches, &s.Wins, &s.Losses, &s.Ties, &s.ScoreRate, &s.WinRate); err != nil {
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

// MatchupStats returns directional archetype-vs-archetype results based on
// actual pairings, not final placement proxies.
//
// Mirror matches (archetype_id == opponent_archetype_id) are a special
// case worth understanding before trusting this endpoint for them: because
// a mirror match's two "directed" rows (A's perspective and B's, where
// A == B) both land in the *same* group, every mirror match contributes
// exactly one win and one loss to that one bucket. win_rate/score_rate for
// any mirror row are therefore mathematically guaranteed to come out at
// 0.5, regardless of what actually happened -- they carry no information.
// matches/wins/losses/ties are still real counts (useful for e.g. "how
// often do mirrors even happen" or draw-rate in the mirror), so rather than
// hide the row entirely, we null out just the two derived rate columns for
// the mirror case so it can't be mistaken for a real 50/50 signal.
func (h *Handler) MatchupStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	metaID := q.Get("meta_id")
	if metaID == "" {
		writeError(w, http.StatusBadRequest, "meta_id is required")
		return
	}

	// default min matches is 20 now
	minMatches := 20
	if v := q.Get("min_matches"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 1 {
		writeError(w, http.StatusBadRequest, "min_matches must be a positive integer")
		return
		}
		minMatches = parsed
	}

	archetypeID := q.Get("archetype_id")
	includeMirrors := q.Get("include_mirrors") == "true"

	// pagination
	page := 1
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
		page = p
		}
	}
	pageSize := 20
	// allow explicit page_size but cap it to 100
	if v := q.Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 {
		if ps > 100 {
			ps = 100
		}
		pageSize = ps
		}
	}
	offset := (page - 1) * pageSize

	// try cache first if redis configured
	cacheKey := fmt.Sprintf("matchups:%s:%s:%d:%t:%d:%d", metaID, archetypeID, minMatches, includeMirrors, page, pageSize)
	if h.Redis != nil {
		if data, err := h.Redis.Get(ctx, cacheKey).Bytes(); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
		}
	}

	countQuery := `
		SELECT COUNT(*) FROM matchups_mv
		WHERE meta_id = $1::uuid
		  AND ($2 = '' OR archetype_id = NULLIF($2,'')::bigint OR opponent_archetype_id = NULLIF($2,'')::bigint)
		  AND ($3 OR archetype_id <> opponent_archetype_id)
		  AND matches >= $4
	`
	var total int
	if err := h.DB.QueryRow(ctx, countQuery, metaID, archetypeID, includeMirrors, minMatches).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "counting matchups: "+err.Error())
		return
	}

	query := `
		SELECT archetype_id, archetype_name, archetype_slug,
		       opponent_archetype_id, opponent_name, opponent_slug,
		       matches, wins, losses, ties, score_rate, win_rate
		FROM matchups_mv
		WHERE meta_id = $1::uuid
		  AND ($2 = '' OR archetype_id = NULLIF($2,'')::bigint OR opponent_archetype_id = NULLIF($2,'')::bigint)
		  AND ($3 OR archetype_id <> opponent_archetype_id)
		  AND matches >= $4
		ORDER BY matches DESC, archetype_name ASC, opponent_name ASC
		LIMIT $5 OFFSET $6
	`

	rows, err := h.DB.Query(ctx, query, metaID, archetypeID, includeMirrors, minMatches, pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying matchup stats: "+err.Error())
		return
	}
	defer rows.Close()

	type matchupStat struct {
		Archetype struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
		} `json:"archetype"`
		Opponent struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
		} `json:"opponent"`
		Matches   int      `json:"matches"`
		Wins      int      `json:"wins"`
		Losses    int      `json:"losses"`
		Ties      int      `json:"ties"`
		ScoreRate *float64 `json:"score_rate"`
		WinRate   *float64 `json:"win_rate"`
	}

	stats := []matchupStat{}
	for rows.Next() {
		var s matchupStat
		if err := rows.Scan(
		&s.Archetype.ID, &s.Archetype.Name, &s.Archetype.Slug,
		&s.Opponent.ID, &s.Opponent.Name, &s.Opponent.Slug,
		&s.Matches, &s.Wins, &s.Losses, &s.Ties, &s.ScoreRate, &s.WinRate,
		); err != nil {
		writeError(w, http.StatusInternalServerError, "scanning matchup stat: "+err.Error())
		return
		}
		stats = append(stats, s)
	}

	// marshal response object and set cache if available
	// compute pagination helpers
	totalPages := 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	prevPage := 0
	if page > 1 {
		prevPage = page - 1
	}
	nextPage := 0
	if page < totalPages {
		nextPage = page + 1
	}

	// construct relative next/prev URLs
	basePath := r.URL.Path
	q := r.URL.Query()
	prevURLStr := ""
	if prevPage > 0 {
		q.Set("page", strconv.Itoa(prevPage))
		prevURLStr = basePath + "?" + q.Encode()
		q.Set("page", strconv.Itoa(page))
	}
	nextURLStr := ""
	if nextPage > 0 {
		q.Set("page", strconv.Itoa(nextPage))
		nextURLStr = basePath + "?" + q.Encode()
		q.Set("page", strconv.Itoa(page))
	}

	respObj := map[string]any{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"prev_page":   prevPage,
		"next_page":   nextPage,
		"prev_url":    prevURLStr,
		"next_url":    nextURLStr,
		"items":       stats,
	}
	b, err := json.Marshal(respObj)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshalling response: "+err.Error())
		return
	}
	if h.Redis != nil {
		ttl := 60 // default
		if s := os.Getenv("MATCHUP_CACHE_TTL_SECONDS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			ttl = v
		}
		}
		_ = h.Redis.Set(ctx, cacheKey, b, time.Duration(ttl)*time.Second)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
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
