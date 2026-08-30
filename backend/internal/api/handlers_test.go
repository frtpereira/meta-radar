package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/frtpereira/meta-radar/internal/ingest"
	"github.com/frtpereira/meta-radar/internal/limitless"
	"github.com/frtpereira/meta-radar/internal/models"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tournamentsResponse struct {
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
	PrevPage   int                 `json:"prev_page"`
	NextPage   int                 `json:"next_page"`
	PrevURL    string              `json:"prev_url"`
	NextURL    string              `json:"next_url"`
	Items      []models.Tournament `json:"items"`
}

type matchupStatBody struct {
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

func TestWriteHelpers(t *testing.T) {
	t.Run("write json", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeJSON(rr, http.StatusCreated, map[string]string{"status": "ok"})
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
	})

	t.Run("write error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeError(rr, http.StatusBadRequest, "bad input")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.JSONEq(t, `{"error":"bad input"}`, rr.Body.String())
	})
}

func TestHealth(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	res := rr.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestListTournaments(t *testing.T) {
	t.Run("defaults and ignored filters", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "", 20, 0).WillReturnRows(pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"}))

		h := &Handler{DB: mock}
		req := httptest.NewRequest(http.MethodGet, "/api/tournaments?min_players=nope&source=invalid&date_from=bad&date_to=bad&page=0&page_size=-5", nil)
		rr := httptest.NewRecorder()
		h.ListTournaments(rr, req)

		resp := decodeBody[tournamentsResponse](t, rr)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 20, resp.PageSize)
		assert.Equal(t, 1, resp.TotalPages)
		assert.Zero(t, resp.PrevPage)
		assert.Zero(t, resp.NextPage)
		assert.Empty(t, resp.Items)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filters pagination and last page", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		dateTo := time.Date(2026, 1, 31, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(64, "STANDARD", "meta-1", boolPtrArg(false), timePtrArg(dateFrom), timePtrArg(dateTo), "dragapult-ex", "cup", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(200))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t`).WithArgs(64, "STANDARD", "meta-1", boolPtrArg(false), timePtrArg(dateFrom), timePtrArg(dateTo), "dragapult-ex", "cup", "", 100, 100).WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"}).
				AddRow("t1", "Cup", "PTCG", "STANDARD", ptrString("meta-1"), ptrString("2026 Meta"), time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC), 128, false, true, ptrString("League"), ptrString("Dragapult ex")),
		)

		h := &Handler{DB: mock}
		req := httptest.NewRequest(http.MethodGet, "/api/tournaments?min_players=64&format=STANDARD&meta_id=meta-1&source=offline&date_from=2026-01-01&date_to=2026-01-31&winner_archetype=dragapult-ex&event_name=cup&page=2&page_size=150", nil)
		rr := httptest.NewRecorder()
		h.ListTournaments(rr, req)

		resp := decodeBody[tournamentsResponse](t, rr)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, 2, resp.Page)
		assert.Equal(t, 100, resp.PageSize)
		assert.Equal(t, 2, resp.TotalPages)
		assert.Equal(t, 1, resp.PrevPage)
		assert.Zero(t, resp.NextPage)
		assert.Contains(t, resp.PrevURL, "page=1")
		assert.Empty(t, resp.NextURL)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "t1", resp.Items[0].ID)
		assert.Equal(t, "meta-1", *resp.Items[0].MetaID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sorts by an allowed column and direction", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t.*ORDER BY t\.players ASC`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "", 20, 0).WillReturnRows(pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"}))

		h := &Handler{DB: mock}
		req := httptest.NewRequest(http.MethodGet, "/api/tournaments?sort_by=players&sort_dir=asc", nil)
		rr := httptest.NewRecorder()
		h.ListTournaments(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sorting by winner archetype always breaks ties by date", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t.*ORDER BY w\.archetype_name ASC NULLS LAST, t\.date DESC`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "", 20, 0).WillReturnRows(pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"}))

		h := &Handler{DB: mock}
		req := httptest.NewRequest(http.MethodGet, "/api/tournaments?sort_by=winner_archetype&sort_dir=asc", nil)
		rr := httptest.NewRecorder()
		h.ListTournaments(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ignores an unrecognized sort_by and falls back to date desc", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t.*ORDER BY t\.date DESC`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "", 20, 0).WillReturnRows(pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"}))

		h := &Handler{DB: mock}
		req := httptest.NewRequest(http.MethodGet, "/api/tournaments?sort_by=name", nil)
		rr := httptest.NewRecorder()
		h.ListTournaments(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database errors", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "").WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		countReq := httptest.NewRequest(http.MethodGet, "/api/tournaments", nil)
		countRR := httptest.NewRecorder()
		h.ListTournaments(countRR, countReq)
		assert.Equal(t, http.StatusInternalServerError, countRR.Code)
		assert.Contains(t, countRR.Body.String(), "counting tournaments")

		mock = newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t`).WithArgs(0, "", "", nilArg(), nilArg(), nilArg(), "", "", "", 20, 0).WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"}).
				AddRow("t1", "Cup", "PTCG", "STANDARD", ptrString("meta-1"), ptrString("Meta"), time.Now(), "bad-players", false, true, ptrString("Org"), ptrString("Winner")),
		)
		h = &Handler{DB: mock}
		scanReq := httptest.NewRequest(http.MethodGet, "/api/tournaments", nil)
		scanRR := httptest.NewRecorder()
		h.ListTournaments(scanRR, scanReq)
		assert.Equal(t, http.StatusInternalServerError, scanRR.Code)
		assert.Contains(t, scanRR.Body.String(), "scanning tournament")
	})
}

func TestTournamentDetail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		when := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*WHERE t\.id = \$1`).WithArgs("t1").WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name"}).
				AddRow("t1", "Regional", "PTCG", "STANDARD", ptrString("meta-1"), ptrString("Meta"), when, 256, true, true, ptrString("League")),
		)
		mock.ExpectQuery(`(?s)SELECT s\.standing, s\.wins, s\.losses, s\.ties.*FROM standings s`).WithArgs("t1").WillReturnRows(
			pgxmock.NewRows([]string{"standing", "wins", "losses", "ties", "player_id", "player_name", "decklist_id", "archetype_id", "archetype_name", "archetype_slug"}).
				AddRow(1, 9, 1, 0, "p1", "Alice", ptrInt64(10), ptrInt64(20), ptrString("Dragapult ex"), ptrString("dragapult-ex")).
				AddRow(0, 4, 3, 0, "p2", "Bob", nil, nil, nil, nil),
		)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/tournaments/t1", nil), "id", "t1")
		rr := httptest.NewRecorder()
		h.TournamentDetail(rr, req)

		var resp struct {
			ID        string `json:"id"`
			Standings []struct {
				PlayerID      string  `json:"player_id"`
				DecklistID    *int64  `json:"decklist_id"`
				ArchetypeName *string `json:"archetype_name"`
			} `json:"standings"`
		}
		resp = decodeBody[struct {
			ID        string `json:"id"`
			Standings []struct {
				PlayerID      string  `json:"player_id"`
				DecklistID    *int64  `json:"decklist_id"`
				ArchetypeName *string `json:"archetype_name"`
			} `json:"standings"`
		}](t, rr)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "t1", resp.ID)
		require.Len(t, resp.Standings, 2)
		assert.Equal(t, "p1", resp.Standings[0].PlayerID)
		assert.Equal(t, int64(10), *resp.Standings[0].DecklistID)
		assert.Equal(t, "Bob", map[string]string{"Bob": "Bob"}["Bob"])
		assert.Nil(t, resp.Standings[1].DecklistID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*WHERE t\.id = \$1`).WithArgs("missing").WillReturnError(pgx.ErrNoRows)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/tournaments/missing", nil), "id", "missing")
		rr := httptest.NewRecorder()
		h.TournamentDetail(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "tournament not found")
	})

	t.Run("standings query error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*WHERE t\.id = \$1`).WithArgs("t1").WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name"}).
				AddRow("t1", "Regional", "PTCG", "STANDARD", nil, nil, time.Now(), 64, true, true, nil),
		)
		mock.ExpectQuery(`(?s)SELECT s\.standing, s\.wins, s\.losses, s\.ties.*FROM standings s`).WithArgs("t1").WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/tournaments/t1", nil), "id", "t1")
		rr := httptest.NewRecorder()
		h.TournamentDetail(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "querying standings")
	})
}

func TestListMetas(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		end := start.Add(7 * 24 * time.Hour)
		mock.ExpectQuery(`SELECT id, name, format_code, starts_at, ends_at FROM metas ORDER BY starts_at DESC`).WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "format_code", "starts_at", "ends_at"}).
				AddRow("m1", "Meta 1", "STANDARD", start, ptrTime(end)).
				AddRow("m2", "Meta 2", "EXPANDED", start.Add(-24*time.Hour), nil),
		)

		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		h.ListMetas(rr, httptest.NewRequest(http.MethodGet, "/api/metas", nil))

		metas := decodeBody[[]models.Meta](t, rr)
		require.Len(t, metas, 2)
		assert.Equal(t, "m1", metas[0].ID)
		assert.NotNil(t, metas[0].EndsAt)
		assert.Nil(t, metas[1].EndsAt)
	})

	t.Run("scan error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, name, format_code, starts_at, ends_at FROM metas ORDER BY starts_at DESC`).WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "format_code", "starts_at", "ends_at"}).
				AddRow("m1", "Meta 1", "STANDARD", "bad-time", nil),
		)

		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		h.ListMetas(rr, httptest.NewRequest(http.MethodGet, "/api/metas", nil))
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "scanning meta")
	})
}

func TestArchetypeStats(t *testing.T) {
	for _, target := range []string{"/api/archetypes/stats", "/api/archetypes/stats?meta_id="} {
		t.Run("missing meta id "+target, func(t *testing.T) {
			h := &Handler{}
			rr := httptest.NewRecorder()
			h.ArchetypeStats(rr, httptest.NewRequest(http.MethodGet, target, nil))
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "meta_id is required")
		})
	}

	mock := newMockDB(t)
	defer mock.Close()
	mock.ExpectQuery(`(?s)WITH sides AS.*ORDER BY deck_count DESC`).WithArgs("meta-1").WillReturnRows(
		pgxmock.NewRows([]string{"id", "name", "slug", "deck_count", "avg_standing", "drop_count", "matches", "wins", "losses", "ties", "score_rate", "win_rate"}).
			AddRow(int64(1), "Dragapult ex", "dragapult-ex", 12, ptrFloat64(3.5), 1, 20, 12, 6, 2, ptrFloat64(0.65), ptrFloat64(0.6667)),
	)

	h := &Handler{DB: mock}
	rr := httptest.NewRecorder()
	h.ArchetypeStats(rr, httptest.NewRequest(http.MethodGet, "/api/archetypes/stats?meta_id=meta-1", nil))
	var resp []struct {
		ID        int64    `json:"id"`
		Avg       *float64 `json:"avg_standing"`
		ScoreRate *float64 `json:"score_rate"`
	}
	resp = decodeBody[[]struct {
		ID        int64    `json:"id"`
		Avg       *float64 `json:"avg_standing"`
		ScoreRate *float64 `json:"score_rate"`
	}](t, rr)
	require.Len(t, resp, 1)
	assert.Equal(t, int64(1), resp[0].ID)
	assert.InDelta(t, 3.5, *resp[0].Avg, 0.001)
	assert.InDelta(t, 0.65, *resp[0].ScoreRate, 0.001)

	t.Run("query error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)WITH sides AS.*ORDER BY deck_count DESC`).WithArgs("meta-1").WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		h.ArchetypeStats(rr, httptest.NewRequest(http.MethodGet, "/api/archetypes/stats?meta_id=meta-1", nil))
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "querying archetype stats")
	})
}

func TestArchetypeDetail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		computedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
		coreCards, _ := json.Marshal([]models.Card{{Name: "Rare Candy", Count: 4, Category: "trainer"}})
		mock.ExpectQuery(`SELECT id, meta_id::text, name, slug, core_cards, core_threshold, core_computed_at FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnRows(
			pgxmock.NewRows([]string{"id", "meta_id", "name", "slug", "core_cards", "core_threshold", "core_computed_at"}).
				AddRow(int64(7), "meta-1", "Dragapult ex", "dragapult-ex", coreCards, ptrFloat64(0.7), ptrTime(computedAt)),
		)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeDetail(rr, req)

		var resp struct {
			ID            int64         `json:"id"`
			CoreCards     []models.Card `json:"core_cards"`
			CoreThreshold *float64      `json:"core_threshold"`
		}
		resp = decodeBody[struct {
			ID            int64         `json:"id"`
			CoreCards     []models.Card `json:"core_cards"`
			CoreThreshold *float64      `json:"core_threshold"`
		}](t, rr)
		assert.Equal(t, int64(7), resp.ID)
		require.Len(t, resp.CoreCards, 1)
		assert.Equal(t, "Rare Candy", resp.CoreCards[0].Name)
		assert.InDelta(t, 0.7, *resp.CoreThreshold, 0.001)
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, meta_id::text, name, slug, core_cards, core_threshold, core_computed_at FROM archetypes WHERE id = \$1`).WithArgs("404").WillReturnError(pgx.ErrNoRows)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/404", nil), "id", "404")
		rr := httptest.NewRecorder()
		h.ArchetypeDetail(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "archetype not found")
	})
}

func TestArchetypeVariants(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT d\.core_hash, COUNT\(\*\) AS deck_count.*FROM decklists d`).WithArgs("7").WillReturnRows(
			pgxmock.NewRows([]string{"core_hash", "deck_count", "avg_standing", "drop_count", "sample_decklist_id"}).
				AddRow("hash1", 8, ptrFloat64(4.1), 1, int64(101)),
		)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/variants", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeVariants(rr, req)

		var resp []struct {
			CoreHash  string `json:"core_hash"`
			DeckCount int    `json:"deck_count"`
		}
		resp = decodeBody[[]struct {
			CoreHash  string `json:"core_hash"`
			DeckCount int    `json:"deck_count"`
		}](t, rr)
		require.Len(t, resp, 1)
		assert.Equal(t, "hash1", resp[0].CoreHash)
	})

	t.Run("scan error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT d\.core_hash, COUNT\(\*\) AS deck_count.*FROM decklists d`).WithArgs("7").WillReturnRows(
			pgxmock.NewRows([]string{"core_hash", "deck_count", "avg_standing", "drop_count", "sample_decklist_id"}).
				AddRow("hash1", "bad-count", ptrFloat64(4.1), 1, int64(101)),
		)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/variants", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeVariants(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "scanning variant")
	})
}

func TestMatchupStats(t *testing.T) {
	for _, target := range []string{"/api/matchups/stats", "/api/matchups/stats?meta_id="} {
		t.Run("missing meta id "+target, func(t *testing.T) {
			h := &Handler{}
			rr := httptest.NewRecorder()
			h.MatchupStats(rr, httptest.NewRequest(http.MethodGet, target, nil))
			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
	for _, target := range []string{"/api/matchups/stats?meta_id=meta-1&min_matches=bad", "/api/matchups/stats?meta_id=meta-1&min_matches=-1"} {
		t.Run("invalid min matches "+target, func(t *testing.T) {
			h := &Handler{}
			rr := httptest.NewRecorder()
			h.MatchupStats(rr, httptest.NewRequest(http.MethodGet, target, nil))
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "min_matches must be a positive integer")
		})
	}

	t.Run("nil redis path", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT archetype_id, archetype_name, archetype_slug.*FROM matchups_mv`).WithArgs("meta-1", "10", false, 25).WillReturnRows(
			pgxmock.NewRows([]string{"archetype_id", "archetype_name", "archetype_slug", "opponent_archetype_id", "opponent_name", "opponent_slug", "matches", "wins", "losses", "ties", "score_rate", "win_rate"}).
				AddRow(int64(10), "Dragapult ex", "dragapult-ex", int64(11), "Gardevoir", "gardevoir", 40, 22, 14, 4, ptrFloat64(0.6), ptrFloat64(0.6111)),
		)

		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		h.MatchupStats(rr, httptest.NewRequest(http.MethodGet, "/api/matchups/stats?meta_id=meta-1&archetype_id=10&include_mirrors=false&min_matches=25", nil))
		resp := decodeBody[[]matchupStatBody](t, rr)
		assert.Equal(t, http.StatusOK, rr.Code)
		require.Len(t, resp, 1)
		assert.Equal(t, int64(10), resp[0].Archetype.ID)
	})

	t.Run("redis cache hit", func(t *testing.T) {
		redisServer := miniredis.RunT(t)
		redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
		defer redisClient.Close()
		cached := `[]`
		require.NoError(t, redisClient.Set(context.Background(), "matchups:meta-1::20:true", cached, time.Minute).Err())

		h := &Handler{Redis: redisClient}
		rr := httptest.NewRecorder()
		h.MatchupStats(rr, httptest.NewRequest(http.MethodGet, "/api/matchups/stats?meta_id=meta-1", nil))
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, cached, rr.Body.String())
	})

	t.Run("redis miss populates cache", func(t *testing.T) {
		redisServer := miniredis.RunT(t)
		redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
		defer redisClient.Close()
		t.Setenv("MATCHUP_CACHE_TTL_SECONDS", "120")

		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT archetype_id, archetype_name, archetype_slug.*FROM matchups_mv`).WithArgs("meta-1", "", true, 20).WillReturnRows(
			pgxmock.NewRows([]string{"archetype_id", "archetype_name", "archetype_slug", "opponent_archetype_id", "opponent_name", "opponent_slug", "matches", "wins", "losses", "ties", "score_rate", "win_rate"}).
				AddRow(int64(1), "Dragapult ex", "dragapult-ex", int64(2), "Miraidon", "miraidon", 25, 14, 9, 2, ptrFloat64(0.6), ptrFloat64(0.6087)),
		)

		h := &Handler{DB: mock, Redis: redisClient}
		rr := httptest.NewRecorder()
		h.MatchupStats(rr, httptest.NewRequest(http.MethodGet, "/api/matchups/stats?meta_id=meta-1", nil))
		assert.Equal(t, http.StatusOK, rr.Code)
		stored, err := redisClient.Get(context.Background(), "matchups:meta-1::20:true").Result()
		require.NoError(t, err)
		assert.JSONEq(t, rr.Body.String(), stored)
	})

	t.Run("query error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT archetype_id, archetype_name, archetype_slug.*FROM matchups_mv`).WithArgs("meta-1", "", true, 20).WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		h.MatchupStats(rr, httptest.NewRequest(http.MethodGet, "/api/matchups/stats?meta_id=meta-1", nil))
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "querying matchup stats")
	})
}

func TestArchetypeCardStats(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT COALESCE\(core_cards, '\[\]'::jsonb\) FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnError(pgx.ErrNoRows)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/card-stats", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeCardStats(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("zero decklists", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT COALESCE\(core_cards, '\[\]'::jsonb\) FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"core_cards"}).AddRow([]byte(`[]`)))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM decklists WHERE archetype_id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/card-stats", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeCardStats(rr, req)
		assert.JSONEq(t, `[]`, rr.Body.String())
	})

	t.Run("aggregates usage", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		coreCards, _ := json.Marshal([]models.Card{{Name: "Rare Candy", Set: "SVI", Number: "191", Count: 4, Category: "trainer"}})
		mock.ExpectQuery(`SELECT COALESCE\(core_cards, '\[\]'::jsonb\) FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"core_cards"}).AddRow(coreCards))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM decklists WHERE archetype_id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`(?s)SELECT\s+c->>'name'.*FROM decklists d`).WithArgs("7").WillReturnRows(
			pgxmock.NewRows([]string{"card_name", "card_set", "card_number", "category", "copy_count", "deck_count"}).
				AddRow("Rare Candy", "SVI", "191", "trainer", 4, 2).
				AddRow("Switch", "SVI", "194", "trainer", 1, 1),
		)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/card-stats", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeCardStats(rr, req)

		var resp []struct {
			Name       string             `json:"name"`
			IsCore     bool               `json:"is_core"`
			DeckCount  int                `json:"deck_count"`
			Presence   float64            `json:"presence"`
			ModalCount int                `json:"modal_count"`
			Dist       map[string]float64 `json:"count_distribution"`
		}
		resp = decodeBody[[]struct {
			Name       string             `json:"name"`
			IsCore     bool               `json:"is_core"`
			DeckCount  int                `json:"deck_count"`
			Presence   float64            `json:"presence"`
			ModalCount int                `json:"modal_count"`
			Dist       map[string]float64 `json:"count_distribution"`
		}](t, rr)
		require.Len(t, resp, 2)
		assert.Equal(t, "Rare Candy", resp[0].Name)
		assert.True(t, resp[0].IsCore)
		assert.Equal(t, 2, resp[0].DeckCount)
		assert.InDelta(t, 1.0, resp[0].Presence, 0.001)
		assert.Equal(t, 4, resp[0].ModalCount)
		assert.InDelta(t, 1.0, resp[0].Dist["4"], 0.001)
	})

	t.Run("row iteration error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT COALESCE\(core_cards, '\[\]'::jsonb\) FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"core_cards"}).AddRow([]byte(`[]`)))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM decklists WHERE archetype_id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		rows := pgxmock.NewRows([]string{"card_name", "card_set", "card_number", "category", "copy_count", "deck_count"}).
			AddRow("Rare Candy", "SVI", "191", "trainer", 4, 2)
		rows.RowError(0, assert.AnError)
		mock.ExpectQuery(`(?s)SELECT\s+c->>'name'.*FROM decklists d`).WithArgs("7").WillReturnRows(rows)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/card-stats", nil), "id", "7")
		rr := httptest.NewRecorder()
		h.ArchetypeCardStats(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "scanning card row")
	})
}

func TestPlayerDetail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		when := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
		mock.ExpectQuery(`SELECT id, name FROM players WHERE lower\(name\) = lower\(\$1\)`).
			WithArgs("Ash").
			WillReturnRows(pgxmock.NewRows([]string{"id", "name"}).AddRow("p1", "Ash Ketchum"))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.date, t\.players.*FROM standings s`).
			WithArgs("p1").
			WillReturnRows(
				pgxmock.NewRows([]string{"id", "name", "date", "players", "standing", "decklist_id", "archetype_id", "archetype_name", "archetype_slug"}).
					AddRow("t1", "Regional", when, 256, 1, ptrInt64(10), ptrInt64(20), ptrString("Dragapult ex"), ptrString("dragapult-ex")).
					AddRow("t2", "League Cup", when, 32, 0, nil, nil, nil, nil),
			)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/players/Ash", nil), "nickname", "Ash")
		rr := httptest.NewRecorder()
		h.PlayerDetail(rr, req)

		type historyRow struct {
			TournamentID  string  `json:"tournament_id"`
			EventName     string  `json:"event_name"`
			Placement     int     `json:"placement"`
			DecklistID    *int64  `json:"decklist_id"`
			ArchetypeName *string `json:"archetype_name"`
		}
		resp := decodeBody[struct {
			ID      string       `json:"id"`
			Name    string       `json:"name"`
			History []historyRow `json:"history"`
		}](t, rr)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "p1", resp.ID)
		assert.Equal(t, "Ash Ketchum", resp.Name)
		require.Len(t, resp.History, 2)
		assert.Equal(t, "t1", resp.History[0].TournamentID)
		assert.Equal(t, "Regional", resp.History[0].EventName)
		assert.Equal(t, 1, resp.History[0].Placement)
		assert.Equal(t, int64(10), *resp.History[0].DecklistID)
		assert.Equal(t, "Dragapult ex", *resp.History[0].ArchetypeName)
		assert.Nil(t, resp.History[1].DecklistID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, name FROM players WHERE lower\(name\) = lower\(\$1\)`).
			WithArgs("missing").WillReturnError(pgx.ErrNoRows)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/players/missing", nil), "nickname", "missing")
		rr := httptest.NewRecorder()
		h.PlayerDetail(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "player not found")
	})

	t.Run("player query error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, name FROM players WHERE lower\(name\) = lower\(\$1\)`).
			WithArgs("Ash").WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/players/Ash", nil), "nickname", "Ash")
		rr := httptest.NewRecorder()
		h.PlayerDetail(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "querying player")
	})

	t.Run("history query error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, name FROM players WHERE lower\(name\) = lower\(\$1\)`).
			WithArgs("Ash").
			WillReturnRows(pgxmock.NewRows([]string{"id", "name"}).AddRow("p1", "Ash Ketchum"))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.date, t\.players.*FROM standings s`).
			WithArgs("p1").WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/players/Ash", nil), "nickname", "Ash")
		rr := httptest.NewRecorder()
		h.PlayerDetail(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "querying player history")
	})
}

func TestDecklistDetail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		when := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
		cardsJSON := []byte(`[{"name":"Charizard ex","set":"OBF","number":"125","count":3,"category":"pokemon"}]`)
		mock.ExpectQuery(`(?s)SELECT d\.id, d\.tournament_id, d\.player_id.*FROM decklists d`).
			WithArgs("10").
			WillReturnRows(
				pgxmock.NewRows([]string{"id", "tournament_id", "player_id", "name", "archetype_id", "name", "slug", "cards", "name", "date"}).
					AddRow(int64(10), "t1", "p1", "Ash Ketchum", ptrInt64(20), ptrString("Charizard ex"), ptrString("charizard-ex"), cardsJSON, "Regional", when),
			)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/decklists/10", nil), "id", "10")
		rr := httptest.NewRecorder()
		h.DecklistDetail(rr, req)

		resp := decodeBody[struct {
			ID             int64         `json:"id"`
			TournamentID   string        `json:"tournament_id"`
			TournamentName string        `json:"tournament_name"`
			PlayerName     string        `json:"player_name"`
			ArchetypeName  *string       `json:"archetype_name"`
			Cards          []models.Card `json:"cards"`
		}](t, rr)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, int64(10), resp.ID)
		assert.Equal(t, "t1", resp.TournamentID)
		assert.Equal(t, "Regional", resp.TournamentName)
		assert.Equal(t, "Ash Ketchum", resp.PlayerName)
		assert.Equal(t, "Charizard ex", *resp.ArchetypeName)
		require.Len(t, resp.Cards, 1)
		assert.Equal(t, "Charizard ex", resp.Cards[0].Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT d\.id, d\.tournament_id, d\.player_id.*FROM decklists d`).
			WithArgs("missing").WillReturnError(pgx.ErrNoRows)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/decklists/missing", nil), "id", "missing")
		rr := httptest.NewRecorder()
		h.DecklistDetail(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "decklist not found")
	})

	t.Run("query error", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		mock.ExpectQuery(`(?s)SELECT d\.id, d\.tournament_id, d\.player_id.*FROM decklists d`).
			WithArgs("10").WillReturnError(assert.AnError)

		h := &Handler{DB: mock}
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/decklists/10", nil), "id", "10")
		rr := httptest.NewRecorder()
		h.DecklistDetail(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "querying decklist")
	})
}

func TestLimitlessWebhook(t *testing.T) {
	t.Run("malformed json", func(t *testing.T) {
		h := &Handler{}
		rr := httptest.NewRecorder()
		h.LimitlessWebhook(rr, httptest.NewRequest(http.MethodPost, "/api/webhooks/limitless", strings.NewReader("{")))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("invalid secret", func(t *testing.T) {
		h := &Handler{WebhookSecret: "expected"}
		rr := httptest.NewRecorder()
		h.LimitlessWebhook(rr, httptest.NewRequest(http.MethodPost, "/api/webhooks/limitless", strings.NewReader(`{"secret":"wrong","event":{"tournamentId":"t1"}}`)))
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("missing tournament id", func(t *testing.T) {
		h := &Handler{WebhookSecret: "expected"}
		rr := httptest.NewRecorder()
		h.LimitlessWebhook(rr, httptest.NewRequest(http.MethodPost, "/api/webhooks/limitless", strings.NewReader(`{"secret":"expected","event":{}}`)))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "missing event.tournamentId")
	})

	t.Run("accepted and async sync runs", func(t *testing.T) {
		mock := newMockDB(t)
		defer mock.Close()
		server, details, standings, pairings := newWebhookLimitlessServer(t)
		defer server.Close()
		expectWebhookSync(mock, details, standings, pairings)

		syncer := ingest.NewSyncer(mock, limitless.NewClient(server.URL, ""))
		h := &Handler{Syncer: syncer, WebhookSecret: "expected"}
		body := bytes.NewBufferString(`{"secret":"expected","event":{"tournamentId":"` + details.ID + `"}}`)
		rr := httptest.NewRecorder()
		h.LimitlessWebhook(rr, httptest.NewRequest(http.MethodPost, "/api/webhooks/limitless", body))

		assert.Equal(t, http.StatusAccepted, rr.Code)
		assert.JSONEq(t, `{"status":"received"}`, rr.Body.String())
		require.Eventually(t, func() bool { return mock.ExpectationsWereMet() == nil }, time.Second, 10*time.Millisecond)
	})
}

func newWebhookLimitlessServer(t *testing.T) (*httptest.Server, *limitless.TournamentDetails, []limitless.StandingEntry, []limitless.PairingEntry) {
	t.Helper()
	details := &limitless.TournamentDetails{TournamentSummary: limitless.TournamentSummary{ID: "t-webhook", Game: "PTCG", Format: "STANDARD", Name: "Webhook Cup", Date: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), Players: 64}}
	details.Decklists = true
	details.IsPublic = true
	details.IsOnline = true
	details.Organizer.Name = "Limitless"
	decklist := json.RawMessage(`{"pokemon":[{"name":"Drakloak","set":"TWM","number":"129","count":4}]}`)
	standing := limitless.StandingEntry{Player: "p1", Name: "Alice", Placing: 1, Decklist: decklist}
	standing.Record.Wins = 5
	standing.Deck = &struct {
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Icons []string `json:"icons"`
	}{ID: "dragapult-ex", Name: "Dragapult ex"}
	pairings := []limitless.PairingEntry{{Round: 1, Phase: 1, Table: 1, Winner: json.RawMessage(`"p1"`), Player1: "p1", Player2: "p2"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tournaments/" + details.ID + "/details":
			_ = json.NewEncoder(w).Encode(details)
		case "/tournaments/" + details.ID + "/standings":
			_ = json.NewEncoder(w).Encode([]limitless.StandingEntry{standing})
		case "/tournaments/" + details.ID + "/pairings":
			_ = json.NewEncoder(w).Encode(pairings)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, details, []limitless.StandingEntry{standing}, pairings
}

func expectWebhookSync(mock pgxmock.PgxPoolIface, details *limitless.TournamentDetails, standings []limitless.StandingEntry, pairings []limitless.PairingEntry) {
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO tournaments`).WithArgs(details.ID, details.Name, details.Game, details.Format, details.Date, details.Players, details.IsOnline, details.IsPublic, details.Decklists, details.Organizer.Name, jsonWebhookArgFor(details)).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(`SELECT id::text FROM metas WHERE format_code = \$1 AND ends_at IS NULL`).WithArgs(details.Format).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(ptrString("meta-1")))
	mock.ExpectExec(`UPDATE tournaments SET meta_id = \$1 WHERE id = \$2`).WithArgs("meta-1", details.ID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	for _, entry := range standings {
		mock.ExpectExec(`INSERT INTO players`).WithArgs(entry.Player, entry.Name).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery(`INSERT INTO archetypes`).WithArgs("meta-1", entry.Deck.Name, entry.Deck.ID).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(3)))
		mock.ExpectQuery(`INSERT INTO decklists`).WithArgs(details.ID, entry.Player, ptrInt64(3), limitless.ParsePTCGDecklist(entry.Decklist), entry.Decklist).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(4)))
		mock.ExpectExec(`INSERT INTO standings`).WithArgs(details.ID, entry.Player, entry.Placing, entry.Record.Wins, entry.Record.Losses, entry.Record.Ties, ptrInt64(4)).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	mock.ExpectExec(`DELETE FROM pairings`).WithArgs(details.ID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
	for _, pairing := range pairings {
		mock.ExpectExec(`INSERT INTO players`).WithArgs(pairing.Player1, pairing.Player1).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO players`).WithArgs(pairing.Player2, pairing.Player2).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO pairings`).WithArgs(details.ID, pairing.Phase, pairing.Round, pairing.Table, pairing.Player1, pairing.Player2, pairing.Player1, "win", jsonWebhookArgFor(pairing)).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO sync_log`).WithArgs(details.ID, "webhook", "success", "").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`).WillReturnResult(pgxmock.NewResult("REFRESH", 1))
}

type jsonWebhookArg []byte

func (a jsonWebhookArg) Match(v interface{}) bool {
	raw, ok := v.([]byte)
	if !ok {
		return false
	}
	var want any
	var got any
	if err := json.Unmarshal([]byte(a), &want); err != nil {
		return false
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		return false
	}
	return assert.ObjectsAreEqual(want, got)
}

func jsonWebhookArgFor(v any) jsonWebhookArg {
	b, _ := json.Marshal(v)
	return jsonWebhookArg(b)
}

var _ = regexp.MustCompile
