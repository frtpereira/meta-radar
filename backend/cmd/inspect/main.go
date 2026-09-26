// cmd/inspect is a throwaway diagnostic, not part of the running app: it
// fetches one tournament's standings and pretty-prints the raw `decklist`
// JSON for the first player that has one, so the actual field shape can be
// checked against what internal/limitless/decklist.go assumes. Pass
// --pairings to inspect a real /pairings response instead, to check
// against what internal/limitless/client.go's PairingEntry assumes.
//
// --labs-tournaments and --labs-decklist inspect the (undocumented) labs
// API instead. GET /tournaments' shape is confirmed (see
// limitlesslabs.TournamentListEntry) except for one assumption --
// --labs-tournaments prints each event's id as used (a plain decimal
// string) so that can be checked against what the other labs endpoints
// actually accept. --labs-decklist cross-checks a decklist against a
// standings row by tp_id (see DecklistEntry's doc comment).
//
// Usage:
//
//	go run ./cmd/inspect --tournament=<id>
//	go run ./cmd/inspect --tournament=<id> --pairings
//	go run ./cmd/inspect --labs-tournaments
//	go run ./cmd/inspect --labs-decklist=<eventID>:<tpID>
//
// or, via the built image:
//
//	docker compose run --rm --entrypoint inspect ingest --tournament=<id> [--pairings]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/frtpereira/meta-radar/internal/config"
	"github.com/frtpereira/meta-radar/internal/limitless"
	"github.com/frtpereira/meta-radar/internal/limitlesslabs"
)

func main() {
	tournamentID := flag.String("tournament", "", "Limitless tournament id to inspect")
	inspectPairings := flag.Bool("pairings", false, "inspect /pairings instead of /standings' decklist field")
	labsTournaments := flag.Bool("labs-tournaments", false, "inspect the labs API's GET /tournaments (event list) response")
	labsDecklist := flag.String("labs-decklist", "", "inspect the labs API's GET /decklist response for <eventID>:<tpID>")
	flag.Parse()

	cfg := config.Load()

	if *labsTournaments {
		inspectLabsTournaments(cfg.LabsAPIBase)
		return
	}
	if *labsDecklist != "" {
		inspectLabsDecklist(cfg.LabsAPIBase, *labsDecklist)
		return
	}

	if *tournamentID == "" {
		fmt.Fprintln(os.Stderr, "Usage: inspect --tournament=<id> [--pairings]")
		fmt.Fprintln(os.Stderr, "       inspect --labs-tournaments")
		fmt.Fprintln(os.Stderr, "       inspect --labs-decklist=<eventID>:<tpID>")
		fmt.Fprintln(os.Stderr, "Find an id from GET /tournaments, or the ingest service's logs.")
		os.Exit(1)
	}

	client := limitless.NewClient(cfg.LimitlessAPIBase, cfg.LimitlessAPIKey)

	if *inspectPairings {
		inspectPairingsData(client, *tournamentID)
		return
	}
	inspectDecklist(client, *tournamentID)
}

func inspectDecklist(client *limitless.Client, tournamentID string) {
	standings, err := client.GetStandings(context.Background(), tournamentID)
	if err != nil {
		log.Fatalf("fetching standings: %v", err)
	}

	fmt.Printf("tournament %s: %d standings entries\n\n", tournamentID, len(standings))

	found := false
	for _, entry := range standings {
		if len(entry.Decklist) == 0 || string(entry.Decklist) == "null" {
			continue
		}
		found = true

		fmt.Printf("player: %s (placing %d)\n", entry.Player, entry.Placing)
		if entry.Deck != nil {
			fmt.Printf("auto-categorized as: %s (%s)\n", entry.Deck.Name, entry.Deck.ID)
		}

		var pretty any
		if err := json.Unmarshal(entry.Decklist, &pretty); err != nil {
			fmt.Println("(decklist field is not valid JSON on its own -- printing raw bytes)")
			fmt.Println(string(entry.Decklist))
			break
		}
		b, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println("raw decklist field:")
		fmt.Println(string(b))
		break // one example is enough
	}

	if !found {
		fmt.Println("no player in this tournament has a non-null decklist field.")
		fmt.Println("try a different --tournament id, ideally one where the /details response has \"decklists\": true.")
	}
}

// inspectPairingsData prints the full raw JSON for a handful of pairings,
// deliberately including a decisive win, a drawn/non-decisive result if
// there is one, and a bye if there is one -- since those are exactly the
// three cases internal/ingest's normalizeWinnerPlayerID/
// classifyPairingResult branch on, and each is worth eyeballing separately
// rather than just checking the first row.
func inspectPairingsData(client *limitless.Client, tournamentID string) {
	pairings, err := client.GetPairings(context.Background(), tournamentID)
	if err != nil {
		log.Fatalf("fetching pairings: %v", err)
	}

	fmt.Printf("tournament %s: %d pairings\n\n", tournamentID, len(pairings))
	if len(pairings) == 0 {
		fmt.Println("no pairings returned -- try a different --tournament id.")
		return
	}

	shown := map[string]bool{}
	printOne := func(label string, p limitless.PairingEntry) {
		fmt.Printf("--- %s ---\n", label)
		b, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			fmt.Printf("(failed to marshal: %v)\n\n", err)
			return
		}
		fmt.Println(string(b))
		fmt.Println()
	}

	for _, p := range pairings {
		switch {
		case p.Player2 == "" && !shown["bye"]:
			printOne("bye (player2 empty)", p)
			shown["bye"] = true
		case string(p.Winner) == "null" && p.Player2 != "" && !shown["no-winner"]:
			printOne("no winner recorded (raw winner: null)", p)
			shown["no-winner"] = true
		case p.Player2 != "" && !shown["decisive"]:
			printOne("ordinary pairing", p)
			shown["decisive"] = true
		}
		if len(shown) == 3 {
			break
		}
	}

	fmt.Println("Check the raw `winner` field above against what")
	fmt.Println("internal/ingest/sync.go's normalizeWinnerPlayerID expects:")
	fmt.Println("  - a JSON string matching player1 or player2 -> a win")
	fmt.Println("  - \"\", null, or -1 -> no winner (draw, if player2 is non-empty)")
	fmt.Println("Anything else gets logged as unrecognized and stored as \"unknown\" --")
	fmt.Println("if the real API uses a different sentinel for draws/byes, update")
	fmt.Println("normalizeWinnerPlayerID accordingly.")
}

// inspectLabsTournaments prints the raw GET /tournaments response so its
// real shape can be checked against limitlesslabs.TournamentListEntry's
// "NOT YET VERIFIED" assumptions -- in particular, what field actually
// holds each event's id.
func inspectLabsTournaments(labsAPIBase string) {
	client := limitlesslabs.NewClient(labsAPIBase)
	entries, err := client.ListTournaments(context.Background())
	if err != nil {
		log.Fatalf("fetching /tournaments: %v", err)
	}

	fmt.Printf("%d official tournaments returned\n\n", len(entries))
	for i, e := range entries {
		marker := ""
		if i == len(entries)-1 {
			marker = " <- assumed most recent"
		}
		status := "completed"
		switch {
		case !e.Started:
			status = "not yet started"
		case !e.Completed:
			status = "in progress"
		}
		fmt.Printf("[%d] id=%q season=%d type=%s city=%s country=%s date=%q (%s)%s\n",
			i, e.ID, e.Season, e.Type, e.City, e.Country, e.Date, status, marker)
	}
	if len(entries) > 0 && entries[len(entries)-1].ID == "" {
		fmt.Println("\nWARNING: the last entry's id came back empty -- check the real")
		fmt.Println("response body against TournamentListEntry's UnmarshalJSON.")
	}
}

// inspectLabsDecklist fetches one decklist and prints it -- useful for
// spot-checking a specific player/tournament. See limitlesslabs.DecklistEntry's
// doc comment: playerId expects StandingEntry.TPID, not PlayerID.
func inspectLabsDecklist(labsAPIBase, arg string) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		log.Fatalf("expected --labs-decklist=<eventID>:<tpID>, got %q", arg)
	}
	eventID := parts[0]
	tpID, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Fatalf("tpID %q is not an integer: %v", parts[1], err)
	}

	client := limitlesslabs.NewClient(labsAPIBase)

	standings, err := client.GetStandings(context.Background(), eventID, "MA")
	if err != nil {
		log.Fatalf("fetching standings (for cross-check): %v", err)
	}
	fmt.Println("standings row(s) with this tp_id:")
	found := false
	for _, s := range standings {
		if s.TPID == tpID {
			found = true
			b, _ := json.MarshalIndent(s, "", "  ")
			fmt.Println(string(b))
		}
	}
	if !found {
		fmt.Printf("(no standings row has tp_id=%d in division MA -- try another division, or this id isn't in that space)\n", tpID)
	}
	fmt.Println()

	decklist, err := client.GetDecklist(context.Background(), eventID, tpID)
	if err != nil {
		log.Fatalf("fetching decklist: %v", err)
	}
	fmt.Println("decklist returned for this tpID:")
	b, _ := json.MarshalIndent(decklist, "", "  ")
	fmt.Println(string(b))
	fmt.Println("\nDoes this decklist's archetype match the deck_id/deck_name on the")
	fmt.Println("standings row above?")
}
