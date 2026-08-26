package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/frtpereira/meta-radar/internal/limitless"
	"github.com/frtpereira/meta-radar/internal/models"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOptions(t *testing.T) {
	assert.Equal(t, Options{
		Game:         "PTCG",
		Format:       "STANDARD",
		MinPlayers:   32,
		MaxPages:     5,
		Refresh:      0,
		RequestDelay: 500 * time.Millisecond,
	}, DefaultOptions())
}

func TestNormalizeWinnerPlayerID(t *testing.T) {
	tests := []struct {
		name       string
		raw        json.RawMessage
		player1    string
		player2    string
		winnerID   string
		recognized bool
	}{
		{name: "empty", raw: json.RawMessage(""), player1: "p1", player2: "p2", recognized: true},
		{name: "null", raw: json.RawMessage("null"), player1: "p1", player2: "p2", recognized: true},
		{name: "minus one", raw: json.RawMessage("-1"), player1: "p1", player2: "p2", recognized: true},
		{name: "player1", raw: json.RawMessage(`"p1"`), player1: "p1", player2: "p2", winnerID: "p1", recognized: true},
		{name: "player2", raw: json.RawMessage(`"p2"`), player1: "p1", player2: "p2", winnerID: "p2", recognized: true},
		{name: "other string", raw: json.RawMessage(`"p3"`), player1: "p1", player2: "p2", recognized: false},
		{name: "raw bytes", raw: json.RawMessage(`not-json`), player1: "p1", player2: "p2", recognized: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winnerID, recognized := normalizeWinnerPlayerID(tt.raw, tt.player1, tt.player2)
			assert.Equal(t, tt.winnerID, winnerID)
			assert.Equal(t, tt.recognized, recognized)
		})
	}
}

func TestClassifyPairingResult(t *testing.T) {
	tests := []struct {
		name         string
		winnerPlayer string
		recognized   bool
		player1      string
		player2      string
		want         string
	}{
		{name: "bye with winner", winnerPlayer: "p1", recognized: true, player1: "p1", player2: "", want: "bye"},
		{name: "bye without winner", recognized: true, player1: "p1", player2: "", want: "unknown"},
		{name: "unrecognized", winnerPlayer: "", recognized: false, player1: "p1", player2: "p2", want: "unknown"},
		{name: "decisive result", winnerPlayer: "p2", recognized: true, player1: "p1", player2: "p2", want: "win"},
		{name: "draw", winnerPlayer: "", recognized: true, player1: "p1", player2: "p2", want: "draw"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, classifyPairingResult(tt.winnerPlayer, tt.recognized, tt.player1, tt.player2))
		})
	}
}

func TestPairingAndDetailsJSON(t *testing.T) {
	pairing := limitless.PairingEntry{Round: 1, Phase: 2, Table: 3, Winner: json.RawMessage(`"p1"`), Player1: "p1", Player2: "p2"}
	details := testTournamentDetails("t1")

	var pairingOut limitless.PairingEntry
	require.NoError(t, json.Unmarshal(pairingJSON(pairing), &pairingOut))
	assert.Equal(t, pairing, pairingOut)

	var detailsOut limitless.TournamentDetails
	require.NoError(t, json.Unmarshal(detailsJSON(details), &detailsOut))
	assert.Equal(t, details.ID, detailsOut.ID)
	assert.Equal(t, details.Organizer.Name, detailsOut.Organizer.Name)
}

func TestShouldSync(t *testing.T) {
	t.Run("no existing row", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs("t1").WillReturnError(pgx.ErrNoRows)

		s := &Syncer{DB: mock}
		should, err := s.shouldSync(context.Background(), "t1", time.Minute)
		require.NoError(t, err)
		assert.True(t, should)
	})

	t.Run("existing row with no refresh", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		lastChecked := time.Now().Add(-10 * time.Minute)
		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs("t1").WillReturnRows(pgxmock.NewRows([]string{"last_checked_at"}).AddRow(lastChecked))

		s := &Syncer{DB: mock}
		should, err := s.shouldSync(context.Background(), "t1", 0)
		require.NoError(t, err)
		assert.False(t, should)
	})

	t.Run("existing row older than refresh window", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		lastChecked := time.Now().Add(-2 * time.Hour)
		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs("t1").WillReturnRows(pgxmock.NewRows([]string{"last_checked_at"}).AddRow(lastChecked))

		s := &Syncer{DB: mock}
		should, err := s.shouldSync(context.Background(), "t1", time.Hour)
		require.NoError(t, err)
		assert.True(t, should)
	})

	t.Run("query error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs("t1").WillReturnError(errors.New("db failed"))

		s := &Syncer{DB: mock}
		should, err := s.shouldSync(context.Background(), "t1", time.Hour)
		require.Error(t, err)
		assert.False(t, should)
	})
}

func TestUpsertArchetypeAndDecklist(t *testing.T) {
	mock := newMockPool(t)
	defer mock.Close()
	tx := beginTx(t, mock)
	defer rollbackTx(t, mock, tx)

	mock.ExpectQuery(`INSERT INTO archetypes`).WithArgs("meta-1", "Uncategorized", "uncategorized").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mock.ExpectQuery(`INSERT INTO decklists`).WithArgs("t1", "p1", ptrInt64(7), []models.Card{{Name: "Rare Candy", Count: 4, Category: "trainer"}}, json.RawMessage(`{"pokemon":[]}`)).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(9)))

	s := &Syncer{DB: mock}
	archetypeID, err := s.upsertArchetype(context.Background(), tx, "meta-1", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(7), archetypeID)

	decklistID, err := s.upsertDecklist(context.Background(), tx, "t1", "p1", ptrInt64(archetypeID), []models.Card{{Name: "Rare Candy", Count: 4, Category: "trainer"}}, json.RawMessage(`{"pokemon":[]}`))
	require.NoError(t, err)
	assert.Equal(t, int64(9), decklistID)
}

func TestUpsertStandingEntry(t *testing.T) {
	t.Run("empty player skipped", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		s := &Syncer{DB: mock}
		require.NoError(t, s.upsertStandingEntry(context.Background(), tx, "t1", nil, limitless.StandingEntry{}))
	})

	t.Run("full path", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		decklist := json.RawMessage(`{"pokemon":[{"name":"Drakloak","set":"TWM","number":"129","count":4}]}`)
		entry := limitless.StandingEntry{Player: "p1", Name: "Alice", Placing: 1, Decklist: decklist}
		entry.Record.Wins = 5
		entry.Record.Losses = 1
		entry.Record.Ties = 0
		entry.Deck = &struct {
			ID    string   `json:"id"`
			Name  string   `json:"name"`
			Icons []string `json:"icons"`
		}{ID: "dragapult-ex", Name: "Dragapult ex"}
		metaID := "meta-1"

		mock.ExpectExec(`INSERT INTO players`).WithArgs("p1", "Alice").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery(`INSERT INTO archetypes`).WithArgs("meta-1", "Dragapult ex", "dragapult-ex").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(5)))
		mock.ExpectQuery(`INSERT INTO decklists`).WithArgs("t1", "p1", ptrInt64(5), limitless.ParsePTCGDecklist(decklist), decklist).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(8)))
		mock.ExpectExec(`INSERT INTO standings`).WithArgs("t1", "p1", 1, 5, 1, 0, ptrInt64(8)).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := &Syncer{DB: mock}
		require.NoError(t, s.upsertStandingEntry(context.Background(), tx, "t1", &metaID, entry))
	})

	t.Run("player only without deck or meta", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		entry := limitless.StandingEntry{Player: "p2", Name: "Bob", Placing: 12}
		entry.Record.Wins = 3
		entry.Record.Losses = 3
		entry.Record.Ties = 1

		mock.ExpectExec(`INSERT INTO players`).WithArgs("p2", "Bob").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO standings`).WithArgs("t1", "p2", 12, 3, 3, 1, (*int64)(nil)).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := &Syncer{DB: mock}
		require.NoError(t, s.upsertStandingEntry(context.Background(), tx, "t1", nil, entry))
	})
}

func TestReplacePairings(t *testing.T) {
	mock := newMockPool(t)
	defer mock.Close()
	tx := beginTx(t, mock)
	defer rollbackTx(t, mock, tx)

	pairings := []limitless.PairingEntry{
		{},
		{Round: 1, Phase: 1, Table: 1, Winner: json.RawMessage(`"alice"`), Player1: "alice"},
		{Round: 1, Phase: 1, Table: 2, Winner: json.RawMessage(`null`), Player1: "alice", Player2: "bob"},
		{Round: 2, Phase: 1, Table: 3, Winner: json.RawMessage(`"mystery"`), Player1: "carl", Player2: "dana"},
	}

	mock.ExpectExec(`DELETE FROM pairings`).WithArgs("t1").WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec(`INSERT INTO players`).WithArgs("alice", "alice").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 1, 1, "alice", "", "alice", "bye", jsonArgFor(pairings[1])).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO players`).WithArgs("alice", "alice").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO players`).WithArgs("bob", "bob").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 1, 2, "alice", "bob", "", "draw", jsonArgFor(pairings[2])).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO players`).WithArgs("carl", "carl").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO players`).WithArgs("dana", "dana").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 2, 3, "carl", "dana", "", "unknown", jsonArgFor(pairings[3])).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := &Syncer{DB: mock}
	require.NoError(t, s.replacePairings(context.Background(), tx, "t1", pairings))

	t.Run("delete error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		mock.ExpectExec(`DELETE FROM pairings`).WithArgs("t1").WillReturnError(errors.New("delete failed"))

		s := &Syncer{DB: mock}
		err := s.replacePairings(context.Background(), tx, "t1", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clearing previous pairings")
	})
}

func TestLogSync(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs("t1", "poll", "success", "").WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := &Syncer{DB: mock}
		s.logSync(context.Background(), "t1", "poll", "success", "")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("write error is logged and ignored", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs("t1", "poll", "error", "boom").WillReturnError(errors.New("write failed"))

		s := &Syncer{DB: mock}
		s.logSync(context.Background(), "t1", "poll", "error", "boom")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSyncTournament(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, standings, pairings := newLimitlessServer(t, map[string]int{})
		defer server.Close()

		expectSuccessfulSync(mock, details, standings, pairings)

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		require.NoError(t, s.syncTournament(context.Background(), details.ID))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("details fetch error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, _, _ := newLimitlessServer(t, map[string]int{"details": http.StatusInternalServerError})
		defer server.Close()

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		err := s.syncTournament(context.Background(), details.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching details")
	})

	t.Run("begin error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, _, _ := newLimitlessServer(t, map[string]int{})
		defer server.Close()
		mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		err := s.syncTournament(context.Background(), details.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "beginning transaction")
	})

	t.Run("pairings fetch error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, _, _ := newLimitlessServer(t, map[string]int{"pairings": http.StatusInternalServerError})
		defer server.Close()

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		err := s.syncTournament(context.Background(), details.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching pairings")
	})
}

func TestSyncOne(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, standings, pairings := newLimitlessServer(t, map[string]int{})
		defer server.Close()

		expectSuccessfulSync(mock, details, standings, pairings)
		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs(details.ID, "webhook", "success", "").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`).WillReturnResult(pgxmock.NewResult("REFRESH", 1))

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		require.NoError(t, s.SyncOne(context.Background(), details.ID))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sync failure logs error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, _, _ := newLimitlessServer(t, map[string]int{"details": http.StatusInternalServerError})
		defer server.Close()

		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs(details.ID, "webhook", "error", stringContainsArg("fetching details")).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		err := s.SyncOne(context.Background(), details.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching details")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRun(t *testing.T) {
	t.Run("skips seen tournaments and refreshes view", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/tournaments":
				page := r.URL.Query().Get("page")
				if page == "2" {
					_ = json.NewEncoder(w).Encode([]limitless.TournamentSummary{})
					return
				}
				_ = json.NewEncoder(w).Encode([]limitless.TournamentSummary{
					{ID: "too-small", Name: "Challenge", Players: 31},
					{ID: "already-synced", Name: "Cup", Players: 64},
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs("already-synced").WillReturnRows(pgxmock.NewRows([]string{"last_checked_at"}).AddRow(time.Now()))
		mock.ExpectExec(`REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`).WillReturnResult(pgxmock.NewResult("REFRESH", 1))

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		require.NoError(t, s.Run(context.Background(), Options{Game: "PTCG", MinPlayers: 32, MaxPages: 2}))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("syncs eligible tournaments and logs success", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, standings, pairings := newLimitlessServer(t, map[string]int{})
		defer server.Close()

		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs(details.ID).WillReturnError(pgx.ErrNoRows)
		expectSuccessfulSync(mock, details, standings, pairings)
		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs(details.ID, "poll", "success", "").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`).WillReturnResult(pgxmock.NewResult("REFRESH", 1))

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		require.NoError(t, s.Run(context.Background(), Options{Game: "PTCG", Format: "STANDARD", MinPlayers: 32, MaxPages: 1}))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("listing error", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		err := s.Run(context.Background(), Options{Game: "PTCG", MaxPages: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing tournaments")
	})

	t.Run("sync failure logs error and still refreshes", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, _, _ := newLimitlessServer(t, map[string]int{"details": http.StatusInternalServerError})
		defer server.Close()

		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs(details.ID).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs(details.ID, "poll", "error", stringContainsArg("fetching details")).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`REFRESH MATERIALIZED VIEW CONCURRENTLY matchups_mv`).WillReturnResult(pgxmock.NewResult("REFRESH", 1))

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		require.NoError(t, s.Run(context.Background(), Options{Game: "PTCG", Format: "STANDARD", MinPlayers: 32, MaxPages: 1}))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("context cancellation during request delay", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		server, details, standings, pairings := newLimitlessServer(t, map[string]int{})
		defer server.Close()

		mock.ExpectQuery(`SELECT last_checked_at FROM tournaments WHERE id = \$1`).WithArgs(details.ID).WillReturnError(pgx.ErrNoRows)
		expectSuccessfulSync(mock, details, standings, pairings)
		mock.ExpectExec(`INSERT INTO sync_log`).WithArgs(details.ID, "poll", "success", "").WillReturnResult(pgxmock.NewResult("INSERT", 1))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s := NewSyncer(mock, limitless.NewClient(server.URL, ""))
		err := s.Run(ctx, Options{Game: "PTCG", Format: "STANDARD", MinPlayers: 32, MaxPages: 1, RequestDelay: time.Second})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	return mock
}

func beginTx(t *testing.T, mock pgxmock.PgxPoolIface) pgx.Tx {
	t.Helper()
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)
	return tx
}

func rollbackTx(t *testing.T, mock pgxmock.PgxPoolIface, tx pgx.Tx) {
	t.Helper()
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrString(v string) *string {
	return &v
}

func testTournamentDetails(id string) *limitless.TournamentDetails {
	details := &limitless.TournamentDetails{TournamentSummary: limitless.TournamentSummary{
		ID:      id,
		Game:    "PTCG",
		Format:  "STANDARD",
		Name:    "Regional",
		Date:    time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
		Players: 64,
	}}
	details.Decklists = true
	details.IsPublic = true
	details.IsOnline = true
	details.Organizer.Name = "Organizer"
	return details
}

func testStandings() []limitless.StandingEntry {
	decklist := json.RawMessage(`{"pokemon":[{"name":"Drakloak","set":"TWM","number":"129","count":4}],"trainer":[{"name":"Rare Candy","set":"SVI","number":"191","count":4}]}`)
	entry := limitless.StandingEntry{Player: "p1", Name: "Alice", Placing: 1, Decklist: decklist}
	entry.Record.Wins = 5
	entry.Record.Losses = 0
	entry.Deck = &struct {
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Icons []string `json:"icons"`
	}{ID: "dragapult-ex", Name: "Dragapult ex"}
	return []limitless.StandingEntry{entry}
}

func testPairings() []limitless.PairingEntry {
	return []limitless.PairingEntry{{Round: 1, Phase: 1, Table: 1, Winner: json.RawMessage(`"p1"`), Player1: "p1", Player2: "p2"}}
}

func newLimitlessServer(t *testing.T, statuses map[string]int) (*httptest.Server, *limitless.TournamentDetails, []limitless.StandingEntry, []limitless.PairingEntry) {
	t.Helper()
	details := testTournamentDetails("t-sync")
	standings := testStandings()
	pairings := testPairings()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		write := func(kind string, payload any) {
			if status := statuses[kind]; status != 0 {
				w.WriteHeader(status)
				return
			}
			_ = json.NewEncoder(w).Encode(payload)
		}
		switch {
		case r.URL.Path == "/tournaments":
			_ = json.NewEncoder(w).Encode([]limitless.TournamentSummary{{ID: details.ID, Name: details.Name, Players: details.Players}})
		case r.URL.Path == fmt.Sprintf("/tournaments/%s/details", details.ID):
			write("details", details)
		case r.URL.Path == fmt.Sprintf("/tournaments/%s/standings", details.ID):
			write("standings", standings)
		case r.URL.Path == fmt.Sprintf("/tournaments/%s/pairings", details.ID):
			write("pairings", pairings)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, details, standings, pairings
}

func expectSuccessfulSync(mock pgxmock.PgxPoolIface, details *limitless.TournamentDetails, standings []limitless.StandingEntry, pairings []limitless.PairingEntry) {
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO tournaments`).WithArgs(details.ID, details.Name, details.Game, details.Format, details.Date, details.Players, details.IsOnline, details.IsPublic, details.Decklists, details.Organizer.Name, jsonArgFor(details)).WillReturnResult(pgxmock.NewResult("INSERT", 1))
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
		mock.ExpectExec(`INSERT INTO pairings`).WithArgs(details.ID, pairing.Phase, pairing.Round, pairing.Table, pairing.Player1, pairing.Player2, pairing.Player1, "win", jsonArgFor(pairing)).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	mock.ExpectCommit()
}

type jsonArg []byte

func (a jsonArg) Match(v interface{}) bool {
	raw, ok := v.([]byte)
	if !ok {
		return false
	}
	var wantValue any
	var gotValue any
	if err := json.Unmarshal([]byte(a), &wantValue); err != nil {
		return false
	}
	if err := json.Unmarshal(raw, &gotValue); err != nil {
		return false
	}
	return assert.ObjectsAreEqual(wantValue, gotValue)
}

func jsonArgFor(v any) jsonArg {
	b, _ := json.Marshal(v)
	return jsonArg(b)
}

type stringContainsArg string

func (a stringContainsArg) Match(v interface{}) bool {
	s, ok := v.(string)
	return ok && regexp.MustCompile(regexp.QuoteMeta(string(a))).MatchString(s)
}
