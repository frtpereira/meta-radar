package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"os"

	"github.com/frtpereira/meta-radar/internal/ingest"
	"github.com/frtpereira/meta-radar/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

// HandlerDB is the narrow slice of pgxpool.Pool the handlers actually need,
// so tests can swap in pgxmock without changing production call sites.
type HandlerDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Handler struct {
	DB            HandlerDB
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

// @Summary Health check
// @Description Reports whether the API is up.
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// tournamentSortColumns whitelists the columns ListTournaments can sort by,
// mapping the public sort_by value to a trusted SQL expression -- the
// query string is never interpolated directly, so an unrecognized value
// can't be used to inject arbitrary SQL.
var tournamentSortColumns = map[string]string{
	"date":             "t.date",
	"players":          "t.players",
	"winner_archetype": "w.archetype_name",
}

// ListTournaments supports ?min_players=32&format=STANDARD&meta_id=...&source=online|offline
// &date_from=YYYY-MM-DD&date_to=YYYY-MM-DD&winner_archetype=<slug>&event_name=<substring>
// &organizer_name=<substring>&sort_by=date|players|winner_archetype&sort_dir=asc|desc
// &page=1&page_size=20
// so the frontend can drive the tournament search filters directly via
// query params rather than filtering client-side. source, date_from, date_to,
// winner_archetype, event_name, organizer_name, and sort_by/sort_dir are all
// optional; an empty/absent value leaves that filter/ordering out entirely.
// date_from/date_to are inclusive on both ends. winner_archetype matches
// the slug of the archetype that took 1st place (see the LATERAL join
// below), not the archetype's display name. event_name matches
// case-insensitively anywhere in the tournament name (SQL LIKE, not an
// exact match), and organizer_name does the same against the organizer's
// name (e.g. organizer_name=DOOM to find tournaments run by "Doom").
// sort_by defaults to "date" (descending) when absent or
// not in tournamentSortColumns; sorting by "winner_archetype" always adds
// date (descending) as a secondary key, both to break ties between same-
// archetype winners and to keep undecided winners (NULL) from scattering.
// Results are paginated (see MatchupStats for the same page/page_size
// convention) instead of the old flat LIMIT 200 array response.
//
// @Summary List tournaments
// @Description Lists tournaments, optionally filtered by minimum player count, format, meta, event name, and organizer name, and sorted by date, players, or winner archetype.
// @Tags tournaments
// @Produce json
// @Param min_players query int false "Minimum number of players"
// @Param format query string false "Format code (e.g. STANDARD)"
// @Param meta_id query string false "Meta UUID to filter by"
// @Param event_name query string false "Case-insensitive substring match on the tournament name"
// @Param organizer_name query string false "Case-insensitive substring match on the organizer name"
// @Param sort_by query string false "Sort column: date, players, or winner_archetype (default date)"
// @Param sort_dir query string false "Sort direction: asc or desc (default desc)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size, max 100 (default 20)"
// @Success 200 {object} apidocs.TournamentsResponse
// @Failure 500 {object} map[string]string
// @Router /api/tournaments [get]
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

	var isOnline *bool
	switch q.Get("source") {
	case "online":
		v := true
		isOnline = &v
	case "offline":
		v := false
		isOnline = &v
	}

	var dateFrom, dateTo *time.Time
	if v := q.Get("date_from"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			dateFrom = &parsed
		}
	}
	if v := q.Get("date_to"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			// end-of-day so the filter is inclusive of the whole day
			endOfDay := parsed.Add(24*time.Hour - time.Nanosecond)
			dateTo = &endOfDay
		}
	}

	winnerArchetypeSlug := q.Get("winner_archetype")
	eventName := q.Get("event_name")
	organizerName := q.Get("organizer_name")

	orderColumn, ok := tournamentSortColumns[q.Get("sort_by")]
	if !ok {
		orderColumn = "t.date"
	}
	orderDir := "DESC"
	if q.Get("sort_dir") == "asc" {
		orderDir = "ASC"
	}
	orderClause := orderColumn + " " + orderDir
	if orderColumn == "w.archetype_name" {
		orderClause += " NULLS LAST, t.date DESC"
	}

	// pagination -- same page/page_size convention as MatchupStats
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

	countQuery := `
		SELECT COUNT(*)
		FROM tournaments t
		LEFT JOIN LATERAL (
			SELECT a.slug AS archetype_slug
			FROM standings s
			JOIN decklists d ON d.id = s.decklist_id
			JOIN archetypes a ON a.id = d.archetype_id
			WHERE s.tournament_id = t.id AND s.standing = 1
			LIMIT 1
		) w ON true
		WHERE t.players >= $1
		  AND ($2 = '' OR t.format_code = $2)
		  AND ($3 = '' OR t.meta_id::text = $3)
		  AND ($4::boolean IS NULL OR t.is_online = $4)
		  AND ($5::timestamptz IS NULL OR t.date >= $5)
		  AND ($6::timestamptz IS NULL OR t.date <= $6)
		  AND ($7 = '' OR w.archetype_slug = $7)
		  AND ($8 = '' OR t.name ILIKE '%' || $8 || '%')
		  AND ($9 = '' OR t.organizer_name ILIKE '%' || $9 || '%')`

	var total int
	if err := h.DB.QueryRow(ctx, countQuery, minPlayers, format, metaID, isOnline, dateFrom, dateTo, winnerArchetypeSlug, eventName, organizerName).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "counting tournaments: "+err.Error())
		return
	}

	query := fmt.Sprintf(`
		SELECT t.id, t.name, t.game, t.format_code, t.meta_id, m.name, t.date, t.players, t.is_online, t.has_decklists, t.organizer_name,
		       w.archetype_name
		FROM tournaments t
		LEFT JOIN metas m ON m.id = t.meta_id
		LEFT JOIN LATERAL (
			SELECT a.name AS archetype_name, a.slug AS archetype_slug
			FROM standings s
			JOIN decklists d ON d.id = s.decklist_id
			JOIN archetypes a ON a.id = d.archetype_id
			WHERE s.tournament_id = t.id AND s.standing = 1
			LIMIT 1
		) w ON true
		WHERE t.players >= $1
		  AND ($2 = '' OR t.format_code = $2)
		  AND ($3 = '' OR t.meta_id::text = $3)
		  AND ($4::boolean IS NULL OR t.is_online = $4)
		  AND ($5::timestamptz IS NULL OR t.date >= $5)
		  AND ($6::timestamptz IS NULL OR t.date <= $6)
		  AND ($7 = '' OR w.archetype_slug = $7)
		  AND ($8 = '' OR t.name ILIKE '%%' || $8 || '%%')
		  AND ($9 = '' OR t.organizer_name ILIKE '%%' || $9 || '%%')
		ORDER BY %s
		LIMIT $10 OFFSET $11`, orderClause)

	rows, err := h.DB.Query(ctx, query, minPlayers, format, metaID, isOnline, dateFrom, dateTo, winnerArchetypeSlug, eventName, organizerName, pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying tournaments: "+err.Error())
		return
	}
	defer rows.Close()

	tournaments := []models.Tournament{}
	for rows.Next() {
		var t models.Tournament
		if err := rows.Scan(&t.ID, &t.Name, &t.Game, &t.FormatCode, &t.MetaID, &t.MetaName, &t.Date, &t.Players, &t.IsOnline, &t.HasDecklists, &t.OrganizerName, &t.WinnerArchetype); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning tournament: "+err.Error())
			return
		}
		tournaments = append(tournaments, t)
	}

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

	basePath := r.URL.Path
	respQuery := r.URL.Query()
	prevURLStr := ""
	if prevPage > 0 {
		respQuery.Set("page", strconv.Itoa(prevPage))
		prevURLStr = basePath + "?" + respQuery.Encode()
		respQuery.Set("page", strconv.Itoa(page))
	}
	nextURLStr := ""
	if nextPage > 0 {
		respQuery.Set("page", strconv.Itoa(nextPage))
		nextURLStr = basePath + "?" + respQuery.Encode()
		respQuery.Set("page", strconv.Itoa(page))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"prev_page":   prevPage,
		"next_page":   nextPage,
		"prev_url":    prevURLStr,
		"next_url":    nextURLStr,
		"items":       tournaments,
	})
}

// TournamentDetail returns one tournament's metadata plus its full
// standings, joined with the player, decklist, and archetype behind each
// entry -- everything a tournament page needs for a leaderboard: standing,
// player, archetype, and match record. A standing of 0 means the player
// dropped rather than finished in that position (see standings comment in
// the schema), so those rows are sorted to the end instead of to the top.
//
// @Summary Get tournament detail
// @Description Returns a tournament's metadata plus its full standings (player, archetype, match record).
// @Tags tournaments
// @Produce json
// @Param id path string true "Tournament ID"
// @Success 200 {object} apidocs.TournamentDetail
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tournaments/{id} [get]
func (h *Handler) TournamentDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var t models.Tournament
	err := h.DB.QueryRow(ctx, `
		SELECT t.id, t.name, t.game, t.format_code, t.meta_id, m.name, t.date, t.players, t.is_online, t.has_decklists, t.organizer_name
		FROM tournaments t
		LEFT JOIN metas m ON m.id = t.meta_id
		WHERE t.id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Game, &t.FormatCode, &t.MetaID, &t.MetaName, &t.Date, &t.Players, &t.IsOnline, &t.HasDecklists, &t.OrganizerName)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "tournament not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying tournament: "+err.Error())
		return
	}

	query := `
		SELECT s.standing, s.wins, s.losses, s.ties,
		       p.id, p.name,
		       d.id, a.id, a.name, a.slug
		FROM standings s
		JOIN players p ON p.id = s.player_id
		LEFT JOIN decklists d ON d.id = s.decklist_id
		LEFT JOIN archetypes a ON a.id = d.archetype_id
		WHERE s.tournament_id = $1
		ORDER BY
			CASE WHEN s.standing = 0 THEN 1 ELSE 0 END, -- drops sort last
			s.standing ASC`

	rows, err := h.DB.Query(ctx, query, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying standings: "+err.Error())
		return
	}
	defer rows.Close()

	type standingRow struct {
		Standing      int     `json:"standing"`
		Wins          int     `json:"wins"`
		Losses        int     `json:"losses"`
		Ties          int     `json:"ties"`
		PlayerID      string  `json:"player_id"`
		PlayerName    string  `json:"player_name"`
		DecklistID    *int64  `json:"decklist_id,omitempty"`
		ArchetypeID   *int64  `json:"archetype_id,omitempty"`
		ArchetypeName *string `json:"archetype_name,omitempty"`
		ArchetypeSlug *string `json:"archetype_slug,omitempty"`
	}

	standings := []standingRow{}
	for rows.Next() {
		var s standingRow
		if err := rows.Scan(&s.Standing, &s.Wins, &s.Losses, &s.Ties,
			&s.PlayerID, &s.PlayerName,
			&s.DecklistID, &s.ArchetypeID, &s.ArchetypeName, &s.ArchetypeSlug); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning standing: "+err.Error())
			return
		}
		standings = append(standings, s)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":             t.ID,
		"name":           t.Name,
		"game":           t.Game,
		"format_code":    t.FormatCode,
		"meta_id":        t.MetaID,
		"meta_name":      t.MetaName,
		"date":           t.Date,
		"players":        t.Players,
		"is_online":      t.IsOnline,
		"has_decklists":  t.HasDecklists,
		"organizer_name": t.OrganizerName,
		"standings":      standings,
	})
}

// @Summary List metas
// @Description Lists all known metas (format eras) ordered by start date descending.
// @Tags metas
// @Produce json
// @Success 200 {array} models.Meta
// @Failure 500 {object} map[string]string
// @Router /api/metas [get]
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
//
// @Summary Archetype stats for a meta
// @Description Returns per-archetype play counts, average standing, and win/score rates for a given meta.
// @Tags archetypes
// @Produce json
// @Param meta_id query string true "Meta UUID"
// @Success 200 {array} apidocs.ArchetypeStat
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/archetypes/stats [get]
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
//
// @Summary Get archetype detail
// @Description Returns an archetype's metadata plus its computed core card list.
// @Tags archetypes
// @Produce json
// @Param id path string true "Archetype ID"
// @Success 200 {object} apidocs.ArchetypeDetail
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/archetypes/{id} [get]
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
//
// @Summary List archetype variants
// @Description Groups an archetype's decklists by core_hash into distinct build variants. Requires cmd/cluster to have run.
// @Tags archetypes
// @Produce json
// @Param id path string true "Archetype ID"
// @Success 200 {array} apidocs.Variant
// @Failure 500 {object} map[string]string
// @Router /api/archetypes/{id}/variants [get]
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
// case worth understanding before trusting this endpoint for them: both
// sides of a mirror are the same archetype, so player1/player2 (arbitrary
// table-position labels from the source data, not archetype identity)
// carry no directional meaning. matchups_mv accounts for this by making
// wins and losses both equal the decisive-match count (matches - ties)
// for mirror rows, instead of splitting them by player slot -- every
// decisive mirror match is simultaneously a win and a loss for the one
// archetype involved. win_rate/score_rate are still nulled out for mirror
// rows since a 50/50 rate carries no information either way.
//
// @Summary Archetype matchup stats
// @Description Returns directional archetype-vs-archetype matchup results for a meta. Not paginated -- matchups_mv rows for one meta are bounded by archetype-pair count, small enough for the frontend to sort/paginate client-side.
// @Tags matchups
// @Produce json
// @Param meta_id query string true "Meta UUID"
// @Param min_matches query int false "Minimum number of matches required to include a matchup (default 20)"
// @Param archetype_id query string false "Filter to matchups involving this archetype ID"
// @Param include_mirrors query bool false "Include mirror matchups (default true)"
// @Success 200 {array} apidocs.MatchupStat
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/matchups/stats [get]
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
	// mirrors are included by default; only an explicit "false" excludes them
	includeMirrors := q.Get("include_mirrors") != "false"

	// try cache first if redis configured
	cacheKey := fmt.Sprintf("matchups:%s:%s:%d:%t", metaID, archetypeID, minMatches, includeMirrors)
	if h.Redis != nil {
		if data, err := h.Redis.Get(ctx, cacheKey).Bytes(); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
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
	`

	rows, err := h.DB.Query(ctx, query, metaID, archetypeID, includeMirrors, minMatches)
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

	b, err := json.Marshal(stats)
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

// ArchetypeCardStats returns per-card usage statistics for every decklist in
// an archetype: presence rate (what fraction of decklists include the card),
// the count distribution (×1/×2/×3/×4 split across those decklists), and
// the modal count (the most-played copy count). Cards are flagged is_core
// if they appear in the archetype's stored core_cards list (requires
// cmd/cluster to have been run). Results are sorted by presence DESC.
//
// @Summary Archetype card stats
// @Description Returns per-card usage statistics (presence rate, copy-count distribution, modal count) for every decklist in an archetype.
// @Tags archetypes
// @Produce json
// @Param id path string true "Archetype ID"
// @Success 200 {array} apidocs.CardStat
// @Failure 500 {object} map[string]string
// @Router /api/archetypes/{id}/card-stats [get]
func (h *Handler) ArchetypeCardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	archetypeID := chi.URLParam(r, "id")

	// Fetch core_cards from the archetype so we can flag core vs. optional.
	var coreCardsJSON []byte
	err := h.DB.QueryRow(ctx,
		`SELECT COALESCE(core_cards, '[]'::jsonb) FROM archetypes WHERE id = $1`, archetypeID,
	).Scan(&coreCardsJSON)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "archetype not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying archetype core cards: "+err.Error())
		return
	}

	var coreCards []models.Card
	_ = json.Unmarshal(coreCardsJSON, &coreCards)

	// Build lookup set of core card keys (name|set|number).
	coreSet := make(map[string]bool, len(coreCards))
	for _, c := range coreCards {
		coreSet[fmt.Sprintf("%s|%s|%s", c.Name, c.Set, c.Number)] = true
	}

	// Count total decklists in this archetype (denominator for all rates).
	var totalDecks int
	if err := h.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM decklists WHERE archetype_id = $1`, archetypeID,
	).Scan(&totalDecks); err != nil {
		writeError(w, http.StatusInternalServerError, "counting decklists: "+err.Error())
		return
	}
	if totalDecks == 0 {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	// Expand JSONB cards arrays and count, per (card identity, copy count),
	// how many distinct decklists play exactly that many copies.
	rows, err := h.DB.Query(ctx, `
		SELECT
			c->>'name'                   AS card_name,
			COALESCE(c->>'set', '')      AS card_set,
			COALESCE(c->>'number', '')   AS card_number,
			COALESCE(c->>'category', '') AS category,
			(c->>'count')::int           AS copy_count,
			COUNT(DISTINCT d.id)::int    AS deck_count
		FROM decklists d,
		     jsonb_array_elements(d.cards) AS c
		WHERE d.archetype_id = $1
		  AND (c->>'count') ~ '^[0-9]+$'
		GROUP BY 1, 2, 3, 4, 5
	`, archetypeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying card stats: "+err.Error())
		return
	}
	defer rows.Close()

	type rawRow struct {
		name, set, number, category string
		copyCount, deckCount        int
	}
	var rawRows []rawRow
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.name, &rr.set, &rr.number, &rr.category, &rr.copyCount, &rr.deckCount); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning card row: "+err.Error())
			return
		}
		rawRows = append(rawRows, rr)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "iterating card rows: "+err.Error())
		return
	}

	// Aggregate into one entry per unique card (name+set+number).
	type aggKey struct{ name, set, number string }
	type aggEntry struct {
		name, set, number, category string
		countDist                   map[int]int // copy_count -> deck_count at that count
	}
	agg := map[aggKey]*aggEntry{}
	for _, rr := range rawRows {
		k := aggKey{rr.name, rr.set, rr.number}
		if agg[k] == nil {
			agg[k] = &aggEntry{
				name: rr.name, set: rr.set, number: rr.number, category: rr.category,
				countDist: map[int]int{},
			}
		}
		agg[k].countDist[rr.copyCount] = rr.deckCount
	}

	type cardStatOut struct {
		Name              string             `json:"name"`
		Set               string             `json:"set"`
		Number            string             `json:"number"`
		Category          string             `json:"category"`
		IsCore            bool               `json:"is_core"`
		DeckCount         int                `json:"deck_count"`
		TotalDecklists    int                `json:"total_decklists"`
		Presence          float64            `json:"presence"`
		ModalCount        int                `json:"modal_count"`
		CountDistribution map[string]float64 `json:"count_distribution"`
	}

	result := make([]cardStatOut, 0, len(agg))
	for _, a := range agg {
		// A deck contributes to exactly one count bucket, so summing across
		// all count buckets gives us the card's total presence count.
		presenceCount := 0
		modalCount, modalFreq := 0, 0
		dist := make(map[string]float64, len(a.countDist))
		for cnt, dc := range a.countDist {
			presenceCount += dc
			if dc > modalFreq {
				modalFreq, modalCount = dc, cnt
			}
			dist[strconv.Itoa(cnt)] = float64(dc) / float64(totalDecks)
		}

		coreKey := fmt.Sprintf("%s|%s|%s", a.name, a.set, a.number)
		result = append(result, cardStatOut{
			Name:              a.name,
			Set:               a.set,
			Number:            a.number,
			Category:          a.category,
			IsCore:            coreSet[coreKey],
			DeckCount:         presenceCount,
			TotalDecklists:    totalDecks,
			Presence:          float64(presenceCount) / float64(totalDecks),
			ModalCount:        modalCount,
			CountDistribution: dist,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Presence != result[j].Presence {
			return result[i].Presence > result[j].Presence
		}
		return result[i].Name < result[j].Name
	})

	writeJSON(w, http.StatusOK, result)
}

// PlayerDetail looks up a player by nickname (case-insensitive exact match
// on players.name, since that's the only handle a user searching the
// Players page has) and returns their full tournament history: one row per
// standings entry, newest first, with everything the Players page's table
// needs (placement, event name, date, player count, their own archetype,
// and the decklist behind that finish, if any).
//
// @Summary Get player detail
// @Description Looks up a player by nickname and returns their tournament history (placement, event, date, players, archetype, decklist).
// @Tags players
// @Produce json
// @Param nickname path string true "Player nickname"
// @Success 200 {object} apidocs.PlayerDetail
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/players/{nickname} [get]
func (h *Handler) PlayerDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nickname := chi.URLParam(r, "nickname")

	var playerID, playerName string
	err := h.DB.QueryRow(ctx,
		`SELECT id, name FROM players WHERE lower(name) = lower($1)`, nickname,
	).Scan(&playerID, &playerName)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying player: "+err.Error())
		return
	}

	rows, err := h.DB.Query(ctx, `
		SELECT t.id, t.name, t.date, t.players,
		       s.standing, s.decklist_id,
		       a.id, a.name, a.slug
		FROM standings s
		JOIN tournaments t ON t.id = s.tournament_id
		LEFT JOIN decklists d ON d.id = s.decklist_id
		LEFT JOIN archetypes a ON a.id = d.archetype_id
		WHERE s.player_id = $1
		ORDER BY t.date DESC`, playerID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying player history: "+err.Error())
		return
	}
	defer rows.Close()

	type historyRow struct {
		TournamentID  string    `json:"tournament_id"`
		EventName     string    `json:"event_name"`
		Date          time.Time `json:"date"`
		Players       int       `json:"players"`
		Placement     int       `json:"placement"`
		DecklistID    *int64    `json:"decklist_id,omitempty"`
		ArchetypeID   *int64    `json:"archetype_id,omitempty"`
		ArchetypeName *string   `json:"archetype_name,omitempty"`
		ArchetypeSlug *string   `json:"archetype_slug,omitempty"`
	}

	history := []historyRow{}
	for rows.Next() {
		var hRow historyRow
		if err := rows.Scan(&hRow.TournamentID, &hRow.EventName, &hRow.Date, &hRow.Players,
			&hRow.Placement, &hRow.DecklistID,
			&hRow.ArchetypeID, &hRow.ArchetypeName, &hRow.ArchetypeSlug); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning history: "+err.Error())
			return
		}
		history = append(history, hRow)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      playerID,
		"name":    playerName,
		"history": history,
	})
}

// DecklistDetail returns a single decklist -- the exact cards a player ran
// at a specific tournament -- plus enough context (player, tournament,
// archetype) to render a standalone decklist page.
//
// @Summary Get decklist detail
// @Description Returns a single decklist's cards plus its player, tournament, and archetype context.
// @Tags decklists
// @Produce json
// @Param id path string true "Decklist ID"
// @Success 200 {object} apidocs.DecklistDetail
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/decklists/{id} [get]
func (h *Handler) DecklistDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var (
		d              models.Decklist
		playerName     string
		archetypeName  *string
		archetypeSlug  *string
		tournamentID   string
		tournamentName string
		date           time.Time
		cardsJSON      []byte
	)
	err := h.DB.QueryRow(ctx, `
		SELECT d.id, d.tournament_id, d.player_id, p.name, d.archetype_id, a.name, a.slug,
		       d.cards, t.name, t.date
		FROM decklists d
		JOIN players p ON p.id = d.player_id
		LEFT JOIN archetypes a ON a.id = d.archetype_id
		JOIN tournaments t ON t.id = d.tournament_id
		WHERE d.id = $1`, id,
	).Scan(&d.ID, &tournamentID, &d.PlayerID, &playerName, &d.ArchetypeID, &archetypeName, &archetypeSlug,
		&cardsJSON, &tournamentName, &date)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "decklist not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying decklist: "+err.Error())
		return
	}

	var cards []models.Card
	if len(cardsJSON) > 0 {
		_ = json.Unmarshal(cardsJSON, &cards)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              d.ID,
		"tournament_id":   tournamentID,
		"tournament_name": tournamentName,
		"date":            date,
		"player_id":       d.PlayerID,
		"player_name":     playerName,
		"archetype_id":    d.ArchetypeID,
		"archetype_name":  archetypeName,
		"archetype_slug":  archetypeSlug,
		"cards":           cards,
	})
}

// a sync for that single tournament. Limitless calls this synchronously and
// doesn't document a retry policy, so we acknowledge immediately (202) and
// run the actual sync in the background rather than making Limitless wait
// on our full fetch-standings-and-upsert round trip.
//
// @Summary Limitless tournament webhook
// @Description Receives tournament update events from Limitless TCG and triggers an async sync of the affected tournament.
// @Tags webhooks
// @Accept json
// @Produce json
// @Success 202 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/webhooks/limitless [post]
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
