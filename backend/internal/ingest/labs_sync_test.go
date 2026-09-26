package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/frtpereira/meta-radar/internal/limitlesslabs"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLabsOptions(t *testing.T) {
	assert.Equal(t, LabsOptions{
		SampleSize:         3,
		FormatCode:         "STANDARD",
		OrganizerName:      "Play! Pokémon",
		RequestDelay:       500 * time.Millisecond,
		RecheckWindow:      24 * time.Hour,
		MaxDecklistFetches: 50,
		MinEventDate:       time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
	}, DefaultLabsOptions())
}

func TestSelectRecentEventIDs(t *testing.T) {
	entries := []limitlesslabs.TournamentListEntry{
		{ID: "68", Completed: true},
		{ID: "69", Completed: true},
		{ID: "70", Completed: true},
		{ID: "71", Completed: true},
		{ID: "72", Completed: true},
		{ID: "73", Completed: false}, // scheduled, not yet run
		{ID: "74", Completed: false}, // scheduled, not yet run
	}

	ids, skipped := selectRecentEventIDs(entries, 3)
	assert.Equal(t, []string{"70", "71", "72"}, ids, "should take the 3 most recent *completed* events, ignoring the two scheduled ones at the tail")
	assert.Equal(t, 2, skipped)
}

func TestSelectRecentEventIDsFewerCompletedThanSample(t *testing.T) {
	entries := []limitlesslabs.TournamentListEntry{
		{ID: "1", Completed: true},
		{ID: "2", Completed: false},
	}

	ids, skipped := selectRecentEventIDs(entries, 3)
	assert.Equal(t, []string{"1"}, ids)
	assert.Equal(t, 1, skipped)
}

func TestSyncLabsDivisionSkipsBeforeCutoff(t *testing.T) {
	var standingsCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tournament":
			_, _ = w.Write([]byte(`{"ok":true,"message":{"type":"regional","city":"Utrecht","date":"July 26–27, 2026","round":1,"players":1}}`))
		case "/standings":
			standingsCalled = true
			_, _ = w.Write([]byte(`{"ok":true,"message":[]}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	s := &Syncer{LabsClient: limitlesslabs.NewClient(srv.URL)}
	opts := LabsOptions{
		FormatCode:    "STANDARD",
		OrganizerName: "Play! Pokémon",
		MinEventDate:  time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
	}

	err := s.syncLabsDivision(context.Background(), "0099", "MA", opts)
	assert.ErrorIs(t, err, errLabsEventBeforeCutoff)
	assert.False(t, standingsCalled, "should skip before fetching standings/pairings for a pre-cutoff event")
}

func TestLabsIDHelpers(t *testing.T) {
	assert.Equal(t, "labs-0071-MA", labsTournamentID("0071", "MA"))
	assert.Equal(t, "labs-1010", labsPlayerID(1010))
	assert.Equal(t, "labs-0071", labsEventGroupID("0071"))
}

func TestLabsEventName(t *testing.T) {
	name := ptrString("San Diego Regional Championships")

	tests := []struct {
		name string
		meta *limitlesslabs.TournamentMeta
		div  string
		want string
	}{
		{
			name: "known type with city",
			meta: &limitlesslabs.TournamentMeta{Type: "worlds", City: "San Francisco"},
			div:  "MA",
			want: "Worlds San Francisco (Masters)",
		},
		{
			name: "assumed regional type",
			meta: &limitlesslabs.TournamentMeta{Type: "regional", City: "San Diego"},
			div:  "SR",
			want: "Regional San Diego (Seniors)",
		},
		{
			name: "falls back to explicit name when type/city missing",
			meta: &limitlesslabs.TournamentMeta{Name: name},
			div:  "JR",
			want: "San Diego Regional Championships (Juniors)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, labsEventName(tt.meta, tt.div))
		})
	}
}

func TestUpsertLabsStandingEntry(t *testing.T) {
	t.Run("normal placement", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		entry := limitlesslabs.StandingEntry{PlayerID: 999, TPID: 1, Name: "Alice", Placement: 1, Wins: 10, Losses: 0, Ties: 0}

		mock.ExpectExec(`INSERT INTO players`).WithArgs("labs-1", "Alice").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO standings`).WithArgs("t1", "labs-1", 1, 10, 0, 0).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := &Syncer{DB: mock}
		require.NoError(t, s.upsertLabsStandingEntry(context.Background(), tx, "t1", entry))
	})

	t.Run("dropped player maps to standing 0", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		entry := limitlesslabs.StandingEntry{PlayerID: 998, TPID: 1419, Name: "Arjun Khadse", Placement: 797, Dropped: 1, Wins: 0, Losses: 4, Ties: 0}

		mock.ExpectExec(`INSERT INTO players`).WithArgs("labs-1419", "Arjun Khadse").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO standings`).WithArgs("t1", "labs-1419", 0, 0, 4, 0).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := &Syncer{DB: mock}
		require.NoError(t, s.upsertLabsStandingEntry(context.Background(), tx, "t1", entry))
	})

	t.Run("dqed player maps to standing 0", func(t *testing.T) {
		mock := newMockPool(t)
		defer mock.Close()
		tx := beginTx(t, mock)
		defer rollbackTx(t, mock, tx)

		entry := limitlesslabs.StandingEntry{PlayerID: 997, TPID: 2, Name: "Bob", Placement: 50, DQed: 1}

		mock.ExpectExec(`INSERT INTO players`).WithArgs("labs-2", "Bob").WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO standings`).WithArgs("t1", "labs-2", 0, 0, 0, 0).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := &Syncer{DB: mock}
		require.NoError(t, s.upsertLabsStandingEntry(context.Background(), tx, "t1", entry))
	})
}

func TestReplaceLabsPairings(t *testing.T) {
	mock := newMockPool(t)
	defer mock.Close()
	tx := beginTx(t, mock)
	defer rollbackTx(t, mock, tx)

	rounds := []labsRoundPairings{
		{round: 1, entries: []limitlesslabs.PairingEntry{
			{}, // skipped: no players at all
			{Table: 1, Player1: 1, Player2: 2, Winner: 1},   // decisive win for player1
			{Table: 2, Player1: 3, Player2: 4, Winner: 0},   // draw
			{Table: 3, Player1: 5, Player2: 0, Winner: 5},   // bye
			{Table: 4, Player1: 6, Player2: 7, Winner: 999}, // winner matches neither -- unknown
		}},
	}

	mock.ExpectExec(`DELETE FROM pairings WHERE tournament_id = \$1`).WithArgs("t1").WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 1, 1, ptrString("labs-1"), ptrString("labs-2"), ptrString("labs-1"), "win", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 1, 2, ptrString("labs-3"), ptrString("labs-4"), (*string)(nil), "draw", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 1, 3, ptrString("labs-5"), (*string)(nil), ptrString("labs-5"), "bye", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO pairings`).WithArgs("t1", 1, 1, 4, ptrString("labs-6"), ptrString("labs-7"), (*string)(nil), "unknown", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := &Syncer{DB: mock}
	require.NoError(t, s.replaceLabsPairings(context.Background(), tx, "t1", rounds))
}
