// Package limitless is a thin, typed client over the public Limitless TCG
// API (https://docs.limitlesstcg.com/developer). No API key is required for
// the endpoints used here (tournaments/details/standings); a key is only
// needed for /games/{id}/decks.
package limitless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TournamentSummary is the shape returned by GET /tournaments.
type TournamentSummary struct {
	ID      string    `json:"id"`
	Game    string    `json:"game"`
	Format  string    `json:"format"`
	Name    string    `json:"name"`
	Date    time.Time `json:"date"`
	Players int       `json:"players"`
}

// TournamentDetails is GET /tournaments/{id}/details -- everything from
// TournamentSummary plus organizer/structure info.
type TournamentDetails struct {
	TournamentSummary
	Organizer struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Logo string `json:"logo"`
	} `json:"organizer"`
	Platform  string `json:"platform"`
	Decklists bool   `json:"decklists"`
	IsPublic  bool   `json:"isPublic"`
	IsOnline  bool   `json:"isOnline"`
	Phases    []struct {
		Phase  int    `json:"phase"`
		Type   string `json:"type"`
		Rounds int    `json:"rounds"`
		Mode   string `json:"mode"`
	} `json:"phases"`
	SpecialRules []string `json:"specialRules"`
}

// StandingEntry is one element of GET /tournaments/{id}/standings.
// `Decklist` is intentionally json.RawMessage: its schema is game-specific
// and only partially documented, so we keep the raw bytes for storage and
// parse what we can separately (see ParsePTCGDecklist).
type StandingEntry struct {
	Player   string `json:"player"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	Standing int    `json:"standing"`
	Record   struct {
		Wins   int `json:"wins"`
		Losses int `json:"losses"`
		Ties   int `json:"ties"`
	} `json:"record"`
	Decklist json.RawMessage `json:"decklist"`
	Deck     *struct {
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Icons []string `json:"icons"`
	} `json:"deck"`
	Drop *int `json:"drop"`
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	const maxAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		if c.apiKey != "" {
			req.Header.Set("X-Access-Key", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("requesting %s: %w", u, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			wait := retryAfter(resp.Header, attempt)
			lastErr = fmt.Errorf("rate limited (429) on %s", u)
			if attempt == maxAttempts {
				break
			}
			select {
			case <-time.After(wait):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, u)
		}

		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decoding response from %s: %w", u, err)
		}
		return nil
	}

	return fmt.Errorf("giving up after %d attempts: %w", maxAttempts, lastErr)
}

// retryAfter honors the Retry-After header (seconds) when the API sends
// one; otherwise it falls back to exponential backoff (1s, 2s, 4s, 8s...).
func retryAfter(h http.Header, attempt int) time.Duration {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	backoff := time.Duration(1<<uint(attempt-1)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	return backoff
}

// ListTournaments wraps GET /tournaments. limit/page <= 0 use the API's
// defaults (50, page 1).
func (c *Client) ListTournaments(ctx context.Context, game, format string, limit, page int) ([]TournamentSummary, error) {
	q := url.Values{}
	if game != "" {
		q.Set("game", game)
	}
	if format != "" {
		q.Set("format", format)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}

	var out []TournamentSummary
	if err := c.get(ctx, "/tournaments", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetTournamentDetails(ctx context.Context, id string) (*TournamentDetails, error) {
	var out TournamentDetails
	if err := c.get(ctx, "/tournaments/"+id+"/details", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetStandings(ctx context.Context, id string) ([]StandingEntry, error) {
	var out []StandingEntry
	if err := c.get(ctx, "/tournaments/"+id+"/standings", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
