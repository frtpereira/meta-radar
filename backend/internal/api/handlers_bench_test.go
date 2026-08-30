package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/frtpereira/meta-radar/internal/models"
	pgxmock "github.com/pashagolub/pgxmock/v3"
)

// This file benchmarks the HTTP handlers backing every backend API endpoint.
// Each Benchmark* function measures request parsing, mocked-DB round trips,
// and JSON response encoding via testing.B/`go test -bench`; the mock and
// its expectations are (re)built outside the timed section on every
// iteration so the numbers reflect handler overhead rather than pgxmock
// bookkeeping. Run with:
//
//	go test ./internal/api/... -bench=. -benchmem -run=^$

func tournamentListRows(n int) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name", "winner_archetype"})
	for i := 0; i < n; i++ {
		rows.AddRow("t1", "Cup", "PTCG", "STANDARD", ptrString("meta-1"), ptrString("2026 Meta"), time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC), 128, false, true, ptrString("League"), ptrString("Dragapult ex"))
	}
	return rows
}

func BenchmarkHealth(b *testing.B) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.Health(rr, req)
	}
}

func BenchmarkListTournaments(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/api/tournaments?min_players=64&format=STANDARD&meta_id=meta-1&source=offline&page=1&page_size=20", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM tournaments t`).WithArgs(64, "STANDARD", "meta-1", boolPtrArg(false), nilArg(), nilArg(), "", "").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(20))
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*FROM tournaments t`).WithArgs(64, "STANDARD", "meta-1", boolPtrArg(false), nilArg(), nilArg(), "", "", 20, 0).WillReturnRows(tournamentListRows(20))
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.ListTournaments(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkTournamentDetail(b *testing.B) {
	when := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/tournaments/t1", nil), "id", "t1")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		mock.ExpectQuery(`(?s)SELECT t\.id, t\.name, t\.game.*WHERE t\.id = \$1`).WithArgs("t1").WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "game", "format_code", "meta_id", "meta_name", "date", "players", "is_online", "has_decklists", "organizer_name"}).
				AddRow("t1", "Regional", "PTCG", "STANDARD", ptrString("meta-1"), ptrString("Meta"), when, 256, true, true, ptrString("League")),
		)
		standings := pgxmock.NewRows([]string{"standing", "wins", "losses", "ties", "player_id", "player_name", "decklist_id", "archetype_id", "archetype_name", "archetype_slug"})
		for j := 0; j < 32; j++ {
			standings.AddRow(j, 9, 1, 0, "p1", "Alice", ptrInt64(10), ptrInt64(20), ptrString("Dragapult ex"), ptrString("dragapult-ex"))
		}
		mock.ExpectQuery(`(?s)SELECT s\.standing, s\.wins, s\.losses, s\.ties.*FROM standings s`).WithArgs("t1").WillReturnRows(standings)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.TournamentDetail(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkListMetas(b *testing.B) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/api/metas", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		mock.ExpectQuery(`SELECT id, name, format_code, starts_at, ends_at FROM metas ORDER BY starts_at DESC`).WillReturnRows(
			pgxmock.NewRows([]string{"id", "name", "format_code", "starts_at", "ends_at"}).
				AddRow("m1", "Meta 1", "STANDARD", start, ptrTime(end)).
				AddRow("m2", "Meta 2", "EXPANDED", start.Add(-24*time.Hour), nil),
		)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.ListMetas(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkArchetypeStats(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/stats?meta_id=meta-1", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		rows := pgxmock.NewRows([]string{"id", "name", "slug", "deck_count", "avg_standing", "drop_count", "matches", "wins", "losses", "ties", "score_rate", "win_rate"})
		for j := 0; j < 25; j++ {
			rows.AddRow(int64(j+1), "Dragapult ex", "dragapult-ex", 12, ptrFloat64(3.5), 1, 20, 12, 6, 2, ptrFloat64(0.65), ptrFloat64(0.6667))
		}
		mock.ExpectQuery(`(?s)WITH sides AS.*ORDER BY deck_count DESC`).WithArgs("meta-1").WillReturnRows(rows)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.ArchetypeStats(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkArchetypeDetail(b *testing.B) {
	computedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	coreCards, _ := json.Marshal([]models.Card{{Name: "Rare Candy", Count: 4, Category: "trainer"}})
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7", nil), "id", "7")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		mock.ExpectQuery(`SELECT id, meta_id::text, name, slug, core_cards, core_threshold, core_computed_at FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnRows(
			pgxmock.NewRows([]string{"id", "meta_id", "name", "slug", "core_cards", "core_threshold", "core_computed_at"}).
				AddRow(int64(7), "meta-1", "Dragapult ex", "dragapult-ex", coreCards, ptrFloat64(0.7), ptrTime(computedAt)),
		)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.ArchetypeDetail(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkArchetypeVariants(b *testing.B) {
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/variants", nil), "id", "7")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		rows := pgxmock.NewRows([]string{"core_hash", "deck_count", "avg_standing", "drop_count", "sample_decklist_id"})
		for j := 0; j < 10; j++ {
			rows.AddRow("hash1", 8, ptrFloat64(4.1), 1, int64(101))
		}
		mock.ExpectQuery(`(?s)SELECT d\.core_hash, COUNT\(\*\) AS deck_count.*FROM decklists d`).WithArgs("7").WillReturnRows(rows)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.ArchetypeVariants(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkMatchupStats(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/api/matchups/stats?meta_id=meta-1", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		rows := pgxmock.NewRows([]string{"archetype_id", "archetype_name", "archetype_slug", "opponent_archetype_id", "opponent_name", "opponent_slug", "matches", "wins", "losses", "ties", "score_rate", "win_rate"})
		for j := 0; j < 30; j++ {
			rows.AddRow(int64(10), "Dragapult ex", "dragapult-ex", int64(11), "Gardevoir", "gardevoir", 40, 22, 14, 4, ptrFloat64(0.6), ptrFloat64(0.6111))
		}
		mock.ExpectQuery(`(?s)SELECT archetype_id, archetype_name, archetype_slug.*FROM matchups_mv`).WithArgs("meta-1", "", true, 20).WillReturnRows(rows)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.MatchupStats(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}

func BenchmarkArchetypeCardStats(b *testing.B) {
	coreCards, _ := json.Marshal([]models.Card{{Name: "Rare Candy", Set: "SVI", Number: "191", Count: 4, Category: "trainer"}})
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/archetypes/7/card-stats", nil), "id", "7")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock := newMockDB(b)
		mock.ExpectQuery(`SELECT COALESCE\(core_cards, '\[\]'::jsonb\) FROM archetypes WHERE id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"core_cards"}).AddRow(coreCards))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM decklists WHERE archetype_id = \$1`).WithArgs("7").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(20))
		cardRows := pgxmock.NewRows([]string{"card_name", "card_set", "card_number", "category", "copy_count", "deck_count"})
		for j := 0; j < 15; j++ {
			cardRows.AddRow("Rare Candy", "SVI", "191", "trainer", 4, 2)
		}
		mock.ExpectQuery(`(?s)SELECT\s+c->>'name'.*FROM decklists d`).WithArgs("7").WillReturnRows(cardRows)
		h := &Handler{DB: mock}
		rr := httptest.NewRecorder()
		b.StartTimer()

		h.ArchetypeCardStats(rr, req)

		b.StopTimer()
		mock.Close()
		b.StartTimer()
	}
}
