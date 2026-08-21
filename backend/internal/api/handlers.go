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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
		SELECT t.id, t.name, t.game, t.format_code, t.meta_id, t.date, t.players, t.is_online, t.has_decklists, t.organizer_name,
		       w.archetype_name
		FROM tournaments t
		LEFT JOIN LATERAL (
			SELECT a.name AS archetype_name
			FROM standings s
			JOIN decklists d ON d.id = s.decklist_id
			JOIN archetypes a ON a.id = d.archetype_id
			WHERE s.tournament_id = t.id AND s.standing = 1
			LIMIT 1
		) w ON true
		WHERE t.players >= $1
		  AND ($2 = '' OR t.format_code = $2)
		  AND ($3 = '' OR t.meta_id::text = $3)
		ORDER BY t.date DESC
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
		if err := rows.Scan(&t.ID, &t.Name, &t.Game, &t.FormatCode, &t.MetaID, &t.Date, &t.Players, &t.IsOnline, &t.HasDecklists, &t.OrganizerName, &t.WinnerArchetype); err != nil {
			writeError(w, http.StatusInternalServerError, "scanning tournament: "+err.Error())
			return
		}
		tournaments = append(tournaments, t)
	}

	writeJSON(w, http.StatusOK, tournaments)
}

// TournamentDetail returns one tournament's metadata plus its full
// standings, joined with the player, decklist, and archetype behind each
// entry -- everything a tournament page needs for a leaderboard: standing,
// player, archetype, and match record. A standing of 0 means the player
// dropped rather than finished in that position (see standings comment in
// the schema), so those rows are sorted to the end instead of to the top.
func (h *Handler) TournamentDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var t models.Tournament
	err := h.DB.QueryRow(ctx, `
		SELECT id, name, game, format_code, meta_id, date, players, is_online, has_decklists, organizer_name
		FROM tournaments
		WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Game, &t.FormatCode, &t.MetaID, &t.Date, &t.Players, &t.IsOnline, &t.HasDecklists, &t.OrganizerName)
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
		"date":           t.Date,
		"players":        t.Players,
		"is_online":      t.IsOnline,
		"has_decklists":  t.HasDecklists,
		"organizer_name": t.OrganizerName,
		"standings":      standings,
	})
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
// case worth understanding before trusting this endpoint for them: both
// sides of a mirror are the same archetype, so player1/player2 (arbitrary
// table-position labels from the source data, not archetype identity)
// carry no directional meaning. matchups_mv accounts for this by making
// wins and losses both equal the decisive-match count (matches - ties)
// for mirror rows, instead of splitting them by player slot -- every
// decisive mirror match is simultaneously a win and a loss for the one
// archetype involved. win_rate/score_rate are still nulled out for mirror
// rows since a 50/50 rate carries no information either way.
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
	q = r.URL.Query()
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

// ArchetypeCardStats returns per-card usage statistics for every decklist in
// an archetype: presence rate (what fraction of decklists include the card),
// the count distribution (×1/×2/×3/×4 split across those decklists), and
// the modal count (the most-played copy count). Cards are flagged is_core
// if they appear in the archetype's stored core_cards list (requires
// cmd/cluster to have been run). Results are sorted by presence DESC.
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
