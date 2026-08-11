// cmd/inspect is a throwaway diagnostic, not part of the running app: it
// fetches one tournament's standings and pretty-prints the raw `decklist`
// JSON for the first player that has one, so the actual field shape can be
// checked against what internal/limitless/decklist.go assumes.
//
// Usage:
//
//	go run ./cmd/inspect --tournament=<id>
//
// or, via the built image:
//
//	docker compose run --rm --entrypoint inspect ingest --tournament=<id>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/frtpereira/pokemon-tcg-tracker/internal/config"
	"github.com/frtpereira/pokemon-tcg-tracker/internal/limitless"
)

func main() {
	tournamentID := flag.String("tournament", "", "Limitless tournament id to inspect (required)")
	flag.Parse()

	if *tournamentID == "" {
		fmt.Fprintln(os.Stderr, "Usage: inspect --tournament=<id>")
		fmt.Fprintln(os.Stderr, "Find an id from GET /tournaments, or the ingest service's logs.")
		os.Exit(1)
	}

	cfg := config.Load()
	client := limitless.NewClient(cfg.LimitlessAPIBase, cfg.LimitlessAPIKey)

	standings, err := client.GetStandings(context.Background(), *tournamentID)
	if err != nil {
		log.Fatalf("fetching standings: %v", err)
	}

	fmt.Printf("tournament %s: %d standings entries\n\n", *tournamentID, len(standings))

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
