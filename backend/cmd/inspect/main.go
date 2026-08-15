// cmd/inspect is a throwaway diagnostic, not part of the running app: it
// fetches one tournament's standings and pretty-prints the raw `decklist`
// JSON for the first player that has one, so the actual field shape can be
// checked against what internal/limitless/decklist.go assumes. Pass
// --pairings to inspect a real /pairings response instead, to check
// against what internal/limitless/client.go's PairingEntry assumes.
//
// Usage:
//
//	go run ./cmd/inspect --tournament=<id>
//	go run ./cmd/inspect --tournament=<id> --pairings
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

	"github.com/frtpereira/meta-radar/internal/config"
	"github.com/frtpereira/meta-radar/internal/limitless"
)

func main() {
	tournamentID := flag.String("tournament", "", "Limitless tournament id to inspect (required)")
	inspectPairings := flag.Bool("pairings", false, "inspect /pairings instead of /standings' decklist field")
	flag.Parse()

	if *tournamentID == "" {
		fmt.Fprintln(os.Stderr, "Usage: inspect --tournament=<id> [--pairings]")
		fmt.Fprintln(os.Stderr, "Find an id from GET /tournaments, or the ingest service's logs.")
		os.Exit(1)
	}

	cfg := config.Load()
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
