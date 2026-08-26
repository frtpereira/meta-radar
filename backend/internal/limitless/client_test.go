package limitless

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryAfter(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		attempt int
		want    time.Duration
	}{
		{name: "header seconds", header: "7", attempt: 1, want: 7 * time.Second},
		{name: "missing header", attempt: 3, want: 4 * time.Second},
		{name: "invalid header", header: "abc", attempt: 2, want: 2 * time.Second},
		{name: "cap at thirty seconds", attempt: 8, want: 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.header != "" {
				h.Set("Retry-After", tt.header)
			}
			assert.Equal(t, tt.want, retryAfter(h, tt.attempt))
		})
	}
}

func TestClientGetSetsAPIKeyAndDecodes(t *testing.T) {
	var gotHeader string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Access-Key")
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]TournamentSummary{{ID: "t1", Name: "Cup"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key")
	var out []TournamentSummary
	err := client.get(context.Background(), "/tournaments", url.Values{"page": {"2"}}, &out)

	require.NoError(t, err)
	assert.Equal(t, "secret-key", gotHeader)
	assert.Equal(t, "page=2", gotQuery)
	assert.Len(t, out, 1)
	assert.Equal(t, "t1", out[0].ID)
}

func TestClientGetLeavesAPIKeyUnsetWhenEmpty(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Access-Key")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	var out map[string]string
	require.NoError(t, client.get(context.Background(), "/status", nil, &out))
	assert.Empty(t, gotHeader)
}

func TestClientGetRetriesOn429(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode([]TournamentSummary{{ID: "retry-success"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	var out []TournamentSummary
	start := time.Now()
	err := client.get(context.Background(), "/tournaments", nil, &out)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, time.Second)
	assert.EqualValues(t, 2, attempts.Load())
	assert.Equal(t, "retry-success", out[0].ID)
}

func TestClientGetRetriesOn429WithoutHeader(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode([]TournamentSummary{{ID: "default-backoff"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	var out []TournamentSummary
	start := time.Now()
	err := client.get(context.Background(), "/tournaments", nil, &out)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, time.Second)
	assert.EqualValues(t, 2, attempts.Load())
	assert.Equal(t, "default-backoff", out[0].ID)
}

func TestClientGetHonorsContextCancellationDuringBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	var out []TournamentSummary
	start := time.Now()
	err := client.get(ctx, "/tournaments", nil, &out)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, time.Second)
}

func TestClientGetReturnsStatusAndDecodeErrors(t *testing.T) {
	t.Run("unexpected status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		var out []TournamentSummary
		err := client.get(context.Background(), "/tournaments", nil, &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 500")
	})

	t.Run("malformed json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id":`))
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		var out []TournamentSummary
		err := client.get(context.Background(), "/tournaments", nil, &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding response")
	})

	t.Run("request build error", func(t *testing.T) {
		client := NewClient("://bad-base", "")
		var out []TournamentSummary
		err := client.get(context.Background(), "/tournaments", nil, &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "building request")
	})
}

func TestClientEndpointWrappers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tournaments":
			assert.Equal(t, "game=PTCG&limit=25&page=2", r.URL.RawQuery)
			_ = json.NewEncoder(w).Encode([]TournamentSummary{{ID: "t1", Name: "Cup"}})
		case "/tournaments/t1/details":
			_ = json.NewEncoder(w).Encode(TournamentDetails{TournamentSummary: TournamentSummary{ID: "t1", Name: "Cup"}})
		case "/tournaments/t1/standings":
			_ = json.NewEncoder(w).Encode([]StandingEntry{{Player: "p1", Name: "Alice"}})
		case "/tournaments/t1/pairings":
			_ = json.NewEncoder(w).Encode([]PairingEntry{{Round: 1, Player1: "p1", Player2: "p2"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	tournaments, err := client.ListTournaments(context.Background(), "PTCG", "", 25, 2)
	require.NoError(t, err)
	require.Len(t, tournaments, 1)
	assert.Equal(t, "t1", tournaments[0].ID)

	details, err := client.GetTournamentDetails(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "Cup", details.Name)

	standings, err := client.GetStandings(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "Alice", standings[0].Name)

	pairings, err := client.GetPairings(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "p2", pairings[0].Player2)
}

func TestClientPropagatesTransportErrors(t *testing.T) {
	client := NewClient("http://example.com", "")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}

	var out []TournamentSummary
	err := client.get(context.Background(), "/tournaments", nil, &out)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requesting")
	assert.Contains(t, err.Error(), "boom")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
