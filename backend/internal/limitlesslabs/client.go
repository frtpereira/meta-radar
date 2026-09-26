// Package limitlesslabs is a thin, typed client over the undocumented JSON
// API backing labs.limitlesstcg.com (found via the `data-sveltekit-fetched`
// payloads embedded in that site's page source, at
// mew.limitlesstcg.com/labs/data/tcg/*). Unlike internal/limitless (the
// published Play API for online tournaments), there is no key, no docs,
// and no stability guarantee -- this is scraping the API a webpage happens
// to call, not an integration against a contract. Treat any assumption
// here that isn't backed by a real observed response as suspect, and see
// the per-type comments for which fields are actually verified.
package limitlesslabs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Divisions are the three age divisions every official tournament runs.
// Each is ingested as its own tournament row -- see
// internal/ingest/labs_sync.go.
var Divisions = []string{"MA", "SR", "JR"}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TournamentListEntry is one element of GET /tournaments.
//
// Verified against a live response (2026-09-25): each element is
// {"id":71,"season":2026,"type":"worlds","city":"San Francisco",
// "country":"US","date":"August 28–30, 2026",
// "utc_start":"2026-08-28 16:00:00","started":1,"completed":1}, and the
// array is chronological with the most recent event last. Per product
// knowledge this list already excludes secondary-region events (Korea,
// Malaysia, Philippines, etc.); every entry actually present is official.
//
// One thing is still an assumption rather than a confirmed fact: ID is
// the plain decimal string of the numeric id (e.g. "71"), not zero-padded
// to match the "0071"-style ids used in earlier examples elsewhere in
// this package's comments -- those were most likely just how the id
// happened to be typed into a URL, not a required format, since a
// tournamentId/id query param almost certainly gets parsed as a number
// server-side regardless of leading zeros. If GetTournamentMeta or the
// other endpoints ever reject a plain "71", zero-pad here instead.
//
// Some events (typically the trailing few, scheduled but not yet played)
// have started=0/completed=0 -- see selectRecentEventIDs in the ingest
// package, which filters on Completed before sampling.
type TournamentListEntry struct {
	ID        string
	Season    int
	Type      string
	City      string
	Country   string
	Date      string
	UTCStart  string
	Started   bool
	Completed bool
}

func (e *TournamentListEntry) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID        int    `json:"id"`
		Season    int    `json:"season"`
		Type      string `json:"type"`
		City      string `json:"city"`
		Country   string `json:"country"`
		Date      string `json:"date"`
		UTCStart  string `json:"utc_start"`
		Started   int    `json:"started"`
		Completed int    `json:"completed"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.ID = strconv.Itoa(raw.ID)
	e.Season = raw.Season
	e.Type = raw.Type
	e.City = raw.City
	e.Country = raw.Country
	e.Date = raw.Date
	e.UTCStart = raw.UTCStart
	e.Started = raw.Started != 0
	e.Completed = raw.Completed != 0
	return nil
}

// TournamentMeta is GET /tournament?id={eventID}&division={division}.
// Verified against a live response (tournamentId=0071, division=MA).
type TournamentMeta struct {
	RK9ID       *string `json:"rk9_id"`
	PlaylatamID *string `json:"playlatam_id"`
	Type        string  `json:"type"` // e.g. "worlds"; "regional" is assumed by analogy, not yet observed
	City        string  `json:"city"`
	Name        *string `json:"name"` // often null -- see ParseEventDate/labsEventName in the ingest package
	Date        string  `json:"date"` // e.g. "August 28–30, 2026" -- see ParseEventDate
	Country     string  `json:"country"`
	UpdatedAt   string  `json:"updated_at"`
	Decklists   int     `json:"decklists"` // 0/1: whether the event has decklists enabled at all, not completeness
	Round       int     `json:"round"`     // total Swiss+cut rounds -- how many to pass to GetPairings
	Players     int     `json:"players"`
	PlayersR1   int     `json:"players_r1"`
	Started     int     `json:"started"`
	Completed   int     `json:"completed"`
}

// StandingEntry is one element of GET /standings?tournamentId={eventID}&division={division}.
// Verified against a live response -- returns the full, unpaginated list
// (797 entries for a Worlds-sized field), unlike the paginated table
// rendered on labs.limitlesstcg.com itself.
type StandingEntry struct {
	PlayerID  int    `json:"player_id"` // a larger, not-yet-fully-understood id space -- see GetDecklist's doc comment for the one thing it's used for
	TPID      int    `json:"tp_id"`     // per-tournament sequential id (1..Players); this is what PairingEntry's Player1/Player2/Winner actually reference -- confirmed against a live response (tp_id 1-6 matched pairing table 1204/1205/1206's player1/player2 names exactly), despite what player_id's similarly-plausible name suggests
	Name      string `json:"name"`
	Country   string `json:"country"`
	DropRound *int   `json:"drop_round"`
	Late      int    `json:"late"`
	DQed      int    `json:"dqed"`
	Placement int    `json:"placement"`
	Points    int    `json:"points"`
	Wins      int    `json:"wins"`
	Losses    int    `json:"losses"`
	Ties      int    `json:"ties"`
	Day2      int    `json:"day2"`
	TopCut    int    `json:"topcut"`
	Dropped   int    `json:"dropped"`
	Decklist  int    `json:"decklist"` // 0/1: whether labs has a decklist recorded for this player
	DeckID    string `json:"deck_id"`
	DeckName  string `json:"deck_name"`
	Icons     string `json:"icons"`
}

// PairingEntry is one element of
// GET /pairings?round={n}&tournamentId={eventID}&division={division}.
// Verified against a live response.
//
// Player1/Player2/Winner use the same per-tournament sequential id as
// StandingEntry.TPID (1..Players) -- confirmed by these values falling
// in that range and matching standings rows' names across a sample
// response. P1ID/P2ID below are a *different*, larger id space (matching
// StandingEntry.PlayerID) whose meaning isn't understood yet; they're kept
// for completeness but nothing here relies on them. Player2 == 0 means a
// bye.
type PairingEntry struct {
	Table         int    `json:"table"`
	Completed     int    `json:"completed"`
	Player1       int    `json:"player1"`
	Player2       int    `json:"player2"` // 0 = bye
	Winner        int    `json:"winner"`  // 0 = draw/no winner; otherwise equals Player1 or Player2
	Player1Record string `json:"player1_record"`
	Player2Record string `json:"player2_record"`
	P1Name        string `json:"p1_name"`
	P1Country     string `json:"p1_country"`
	P2Name        string `json:"p2_name"`
	P2Country     string `json:"p2_country"`
	P1DropRound   *int   `json:"p1_drop_round"`
	P2DropRound   *int   `json:"p2_drop_round"`
	P1Dropped     int    `json:"p1_dropped"`
	P2Dropped     int    `json:"p2_dropped"`
	P1Deck        string `json:"p1_deck"`
	P1DeckName    string `json:"p1_deck_name"`
	P1Icons       string `json:"p1_icons"`
	P2Deck        string `json:"p2_deck"`
	P2DeckName    string `json:"p2_deck_name"`
	P2Icons       string `json:"p2_icons"`
	P1ID          int    `json:"p1_id"`
	P2ID          int    `json:"p2_id"`
}

// DecklistEntry is GET /decklist?tournamentId={eventID}&playerId={playerID}.
//
// The shape (pokemon/trainer/energy, each {count,name,set,number}) is
// verified against a live response, and matches
// internal/limitless.ParsePTCGDecklist's expected input exactly -- marshal
// this back to JSON and hand it to that function rather than duplicating a
// parser.
//
// `playerId` expects StandingEntry.TPID, not PlayerID -- confirmed live
// against tournament 71/division MA: playerId=<PlayerID> returned
// {"message":null}, while playerId=<TPID> returned the real decklist
// whose archetype matched that standings row's deck_id/deck_name.
type DecklistEntry struct {
	Pokemon []DecklistCard `json:"pokemon"`
	Trainer []DecklistCard `json:"trainer"`
	Energy  []DecklistCard `json:"energy"`
}

type DecklistCard struct {
	Count  int    `json:"count"`
	Name   string `json:"name"`
	Set    string `json:"set"`
	Number string `json:"number"`
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
		// Identify ourselves -- this is someone else's undocumented API,
		// not a contract we were given, so at minimum it should be
		// obvious in their logs who's calling it.
		req.Header.Set("User-Agent", "meta-radar-ingest (+https://github.com/frtpereira/meta-radar)")

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
			log.Printf("  rate limited (attempt %d/%d), waiting %s before retrying %s", attempt, maxAttempts, wait, u)
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

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading response from %s: %w", u, err)
		}

		var envelope struct {
			OK      bool            `json:"ok"`
			Message json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return fmt.Errorf("decoding envelope from %s: %w", u, err)
		}
		if !envelope.OK {
			return fmt.Errorf("labs API returned ok=false from %s: %s", u, string(envelope.Message))
		}
		if err := json.Unmarshal(envelope.Message, out); err != nil {
			return fmt.Errorf("decoding message from %s: %w", u, err)
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

// ListTournaments wraps GET /tournaments -- see TournamentListEntry's doc
// comment for the verified shape and the one remaining assumption (id
// zero-padding).
func (c *Client) ListTournaments(ctx context.Context) ([]TournamentListEntry, error) {
	var out []TournamentListEntry
	if err := c.get(ctx, "/tournaments", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetTournamentMeta(ctx context.Context, eventID, division string) (*TournamentMeta, error) {
	q := url.Values{"id": {eventID}, "division": {division}}
	var out TournamentMeta
	if err := c.get(ctx, "/tournament", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetStandings(ctx context.Context, eventID, division string) ([]StandingEntry, error) {
	q := url.Values{"tournamentId": {eventID}, "division": {division}}
	var out []StandingEntry
	if err := c.get(ctx, "/standings", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetPairings(ctx context.Context, eventID, division string, round int) ([]PairingEntry, error) {
	q := url.Values{"tournamentId": {eventID}, "division": {division}, "round": {strconv.Itoa(round)}}
	var out []PairingEntry
	if err := c.get(ctx, "/pairings", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDecklist fetches one player's decklist. See DecklistEntry's doc
// comment for the unresolved question of which id space playerID needs to
// be in.
func (c *Client) GetDecklist(ctx context.Context, eventID string, playerID int) (*DecklistEntry, error) {
	q := url.Values{"tournamentId": {eventID}, "playerId": {strconv.Itoa(playerID)}}
	var out DecklistEntry
	if err := c.get(ctx, "/decklist", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

var yearRe = regexp.MustCompile(`\d{4}\s*$`)

// ParseEventDate parses the labs API's human-readable event date range
// (e.g. "August 28–30, 2026", "August 30 – September 1, 2026" for a
// cross-month event, or a single day "August 30, 2026") and returns the
// event's start date. Only the start date is kept -- `tournaments.date`
// is a single timestamp, matching every other Limitless-sourced
// tournament's `date` column.
func ParseEventDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("–", "-", "—", "-").Replace(s) // normalize en/em dash to a plain hyphen

	year := strings.TrimSpace(yearRe.FindString(s))
	if year == "" {
		return time.Time{}, fmt.Errorf("no year found in event date %q", s)
	}

	startPart := strings.TrimSpace(strings.SplitN(s, "-", 2)[0])
	if !strings.Contains(startPart, ",") {
		// Range form ("August 28-30, 2026" or "August 28 - September 1, 2026"):
		// the year trails the *end* of the range, not the start -- borrow it.
		startPart += ", " + year
	}

	t, err := time.Parse("January 2, 2006", startPart)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing %q (from %q): %w", startPart, s, err)
	}
	return t, nil
}
