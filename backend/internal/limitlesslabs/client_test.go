package limitlesslabs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEventDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "same-month range with en dash",
			input: "August 28–30, 2026",
			want:  time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "single day",
			input: "August 30, 2026",
			want:  time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "cross-month range with spaced hyphen",
			input: "August 30 - September 1, 2026",
			want:  time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "cross-month range with en dash",
			input: "August 30 – September 1, 2026",
			want:  time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEventDate(tt.input)
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "want %s, got %s", tt.want, got)
		})
	}
}

func TestParseEventDateRejectsUnparsable(t *testing.T) {
	_, err := ParseEventDate("not a date")
	assert.Error(t, err)
}

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	return NewClient(srv.URL), srv.Close
}

func TestGetTournamentMeta(t *testing.T) {
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tournament", r.URL.Path)
		assert.Equal(t, "0071", r.URL.Query().Get("id"))
		assert.Equal(t, "MA", r.URL.Query().Get("division"))
		w.Write([]byte(`{
			"ok": true,
			"message": {
				"rk9_id": "WCS01wI9lfI55wHtvrdy",
				"type": "worlds",
				"city": "San Francisco",
				"name": null,
				"date": "August 28–30, 2026",
				"country": "US",
				"decklists": 1,
				"round": 16,
				"players": 797,
				"started": 1,
				"completed": 1
			}
		}`))
	})
	defer closeFn()

	meta, err := client.GetTournamentMeta(context.Background(), "0071", "MA")
	require.NoError(t, err)
	assert.Equal(t, "worlds", meta.Type)
	assert.Equal(t, "San Francisco", meta.City)
	assert.Nil(t, meta.Name)
	assert.Equal(t, 16, meta.Round)
	assert.Equal(t, 797, meta.Players)
	assert.Equal(t, 1, meta.Decklists)
}

func TestGetStandings(t *testing.T) {
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/standings", r.URL.Path)
		assert.Equal(t, "0071", r.URL.Query().Get("tournamentId"))
		assert.Equal(t, "MA", r.URL.Query().Get("division"))
		w.Write([]byte(`{
			"ok": true,
			"message": [
				{"player_id": 1, "tp_id": 100, "name": "Alice", "placement": 1, "wins": 10, "losses": 0, "ties": 0, "decklist": 1, "deck_id": "dragapult-ex", "deck_name": "Dragapult"},
				{"player_id": 1419, "tp_id": 559, "name": "Arjun Khadse", "placement": 797, "wins": 0, "losses": 4, "ties": 0, "dropped": 1, "decklist": 1, "deck_id": "dragapult-dusknoir", "deck_name": "Dragapult Dusknoir"}
			]
		}`))
	})
	defer closeFn()

	standings, err := client.GetStandings(context.Background(), "0071", "MA")
	require.NoError(t, err)
	require.Len(t, standings, 2)
	assert.Equal(t, 1, standings[0].PlayerID)
	assert.Equal(t, "Alice", standings[0].Name)
	assert.Equal(t, 1, standings[1].Dropped)
}

func TestGetPairings(t *testing.T) {
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pairings", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("round"))
		assert.Equal(t, "0071", r.URL.Query().Get("tournamentId"))
		assert.Equal(t, "MA", r.URL.Query().Get("division"))
		w.Write([]byte(`{
			"ok": true,
			"message": [
				{"table": 1204, "completed": 1, "player1": 1, "player2": 2, "winner": 1, "p1_name": "A", "p2_name": "B", "p1_id": 54130, "p2_id": 29647}
			]
		}`))
	})
	defer closeFn()

	pairings, err := client.GetPairings(context.Background(), "0071", "MA", 1)
	require.NoError(t, err)
	require.Len(t, pairings, 1)
	assert.Equal(t, 1, pairings[0].Player1)
	assert.Equal(t, 2, pairings[0].Player2)
	assert.Equal(t, 1, pairings[0].Winner)
}

func TestGetDecklist(t *testing.T) {
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/decklist", r.URL.Path)
		assert.Equal(t, "0072", r.URL.Query().Get("tournamentId"))
		assert.Equal(t, "1339", r.URL.Query().Get("playerId"))
		w.Write([]byte(`{
			"ok": true,
			"message": {
				"pokemon": [{"count": 4, "name": "Dreepy", "set": "ASC", "number": "158"}],
				"trainer": [],
				"energy": []
			}
		}`))
	})
	defer closeFn()

	decklist, err := client.GetDecklist(context.Background(), "0072", 1339)
	require.NoError(t, err)
	require.Len(t, decklist.Pokemon, 1)
	assert.Equal(t, "Dreepy", decklist.Pokemon[0].Name)
}

func TestListTournamentsDecodesRealShape(t *testing.T) {
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"message":[
			{"id":70,"season":2026,"type":"international","city":"New Orleans","country":"US","date":"June 12–14, 2026","utc_start":"2026-06-12 14:00:00","started":1,"completed":1},
			{"id":71,"season":2026,"type":"worlds","city":"San Francisco","country":"US","date":"August 28–30, 2026","utc_start":"2026-08-28 16:00:00","started":1,"completed":1},
			{"id":73,"season":2027,"type":"regional","city":"Brisbane","country":"AU","date":"September 26–27, 2026","utc_start":"2026-09-25 22:30:00","started":0,"completed":0}
		]}`))
	})
	defer closeFn()

	entries, err := client.ListTournaments(context.Background())
	require.NoError(t, err)
	require.Len(t, entries, 3)

	assert.Equal(t, "70", entries[0].ID)
	assert.Equal(t, 2026, entries[0].Season)
	assert.Equal(t, "international", entries[0].Type)
	assert.Equal(t, "New Orleans", entries[0].City)
	assert.Equal(t, "US", entries[0].Country)
	assert.Equal(t, "June 12–14, 2026", entries[0].Date)
	assert.Equal(t, "2026-06-12 14:00:00", entries[0].UTCStart)
	assert.True(t, entries[0].Started)
	assert.True(t, entries[0].Completed)

	assert.Equal(t, "71", entries[1].ID)
	assert.Equal(t, "worlds", entries[1].Type)

	assert.Equal(t, "73", entries[2].ID)
	assert.False(t, entries[2].Started, "scheduled-but-not-yet-run events should decode as not started")
	assert.False(t, entries[2].Completed)
}

func TestGetReturnsErrorWhenOKFalse(t *testing.T) {
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok": false, "message": "division not found"}`))
	})
	defer closeFn()

	_, err := client.GetTournamentMeta(context.Background(), "0071", "JR")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "division not found")
}

func TestGetRetriesOn429(t *testing.T) {
	attempts := 0
	client, closeFn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok": true, "message": []}`))
	})
	defer closeFn()

	_, err := client.ListTournaments(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}
