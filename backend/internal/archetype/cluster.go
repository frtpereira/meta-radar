// Package archetype computes, per archetype within a meta, which cards are
// "core" (played across most builds) versus swappable tech -- and hashes
// each decklist's core subset so that different builds of the same
// Limitless-categorized archetype can be told apart as variants.
//
// This is a batch job (see cmd/cluster), not something the ingestion sync
// does per-tournament: knowing what's "core" requires seeing the whole
// population of decklists for an archetype, not just one.
package archetype

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/frtpereira/meta-radar/internal/models"
	"github.com/jackc/pgx/v5"
)

// DefaultCoreThreshold: a card played in at least 70% of an archetype's
// decklists counts as core. Below that, it's treated as a tech/swap slot.
const DefaultCoreThreshold = 0.7

type Clusterer struct {
	DB ClustererDB
}

// ClustererDB narrows database access to the methods clustering actually uses.
type ClustererDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewClusterer(db ClustererDB) *Clusterer {
	return &Clusterer{DB: db}
}

// CardKey canonically identifies a card for frequency counting and hashing.
//
// Trainer and Energy cards are keyed by name alone: the same Trainer or
// Energy reprinted in a different set (e.g. Boss's Orders from PAL and from
// MEG, or Grass Energy from any set) is functionally the same card, so its
// prints must not fragment a core or split a variant. Pokémon stay keyed by
// name + set + number, since different prints of those can differ in ways
// that matter (attacks, abilities, HP).
func CardKey(c models.Card) string {
	switch strings.ToLower(c.Category) {
	case "trainer", "energy":
		return c.Name
	}
	return fmt.Sprintf("%s|%s|%s", c.Name, c.Set, c.Number)
}

// RunForMeta recomputes core cards and core_hash for every archetype in the
// given meta.
func (cl *Clusterer) RunForMeta(ctx context.Context, metaID string, threshold float64) error {
	rows, err := cl.DB.Query(ctx, `SELECT id FROM archetypes WHERE meta_id = $1`, metaID)
	if err != nil {
		return fmt.Errorf("listing archetypes: %w", err)
	}
	var archetypeIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		archetypeIDs = append(archetypeIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range archetypeIDs {
		if err := cl.RunForArchetype(ctx, id, threshold); err != nil {
			return fmt.Errorf("archetype %d: %w", id, err)
		}
	}
	return nil
}

type deckRow struct {
	ID    int64
	Cards []models.Card
}

// RunForArchetype computes the core set for one archetype and hashes every
// one of its decklists against that core.
func (cl *Clusterer) RunForArchetype(ctx context.Context, archetypeID int64, threshold float64) error {
	rows, err := cl.DB.Query(ctx, `SELECT id, cards FROM decklists WHERE archetype_id = $1`, archetypeID)
	if err != nil {
		return fmt.Errorf("loading decklists: %w", err)
	}

	var decks []deckRow
	for rows.Next() {
		var d deckRow
		var raw []byte
		if err := rows.Scan(&d.ID, &raw); err != nil {
			rows.Close()
			return err
		}
		if err := json.Unmarshal(raw, &d.Cards); err != nil {
			rows.Close()
			return fmt.Errorf("decoding cards for decklist %d: %w", d.ID, err)
		}
		decks = append(decks, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(decks) == 0 {
		return nil
	}

	// Pass 1: what fraction of this archetype's decklists play each card?
	presence := map[string]int{} // cardKey -> number of decklists containing it at all
	for _, d := range decks {
		seen := map[string]bool{}
		for _, c := range d.Cards {
			k := CardKey(c)
			if !seen[k] {
				presence[k]++
				seen[k] = true
			}
		}
	}

	total := float64(len(decks))
	core := map[string]bool{}
	for k, n := range presence {
		if float64(n)/total >= threshold {
			core[k] = true
		}
	}

	coreCards := coreCardList(decks, core)
	coreJSON, err := json.Marshal(coreCards)
	if err != nil {
		return err
	}

	tx, err := cl.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE archetypes SET core_cards = $1, core_threshold = $2, core_computed_at = now()
		WHERE id = $3`, coreJSON, threshold, archetypeID); err != nil {
		return fmt.Errorf("updating archetype core: %w", err)
	}

	for _, d := range decks {
		hash := coreHash(d.Cards, core)
		if _, err := tx.Exec(ctx, `UPDATE decklists SET core_hash = $1 WHERE id = $2`, hash, d.ID); err != nil {
			return fmt.Errorf("updating decklist %d core_hash: %w", d.ID, err)
		}
	}

	return tx.Commit(ctx)
}

// coreCardList picks one representative Card per core key. The count is
// whichever total was most common for that card across decklists (so "core"
// reflects a realistic play count, not just presence). Copies of a card that
// share a CardKey but come from different prints within one decklist (e.g.
// 2 Boss's Orders from one set + 2 from another) are summed first. When a
// key has several prints, the most-played one is used for the set/number.
func coreCardList(decks []deckRow, core map[string]bool) []models.Card {
	type printID struct{ set, number string }
	countFreq := map[string]map[int]int{}     // cardKey -> total count -> how many decklists played that total
	printFreq := map[string]map[printID]int{} // cardKey -> print -> total copies played across decklists
	rep := map[string]models.Card{}
	for _, d := range decks {
		totals := map[string]int{}
		for _, c := range d.Cards {
			k := CardKey(c)
			if !core[k] {
				continue
			}
			totals[k] += c.Count
			if printFreq[k] == nil {
				printFreq[k] = map[printID]int{}
			}
			printFreq[k][printID{c.Set, c.Number}] += c.Count
			rep[k] = c
		}
		for k, n := range totals {
			if countFreq[k] == nil {
				countFreq[k] = map[int]int{}
			}
			countFreq[k][n]++
		}
	}

	list := make([]models.Card, 0, len(core))
	for k := range core {
		c := rep[k]

		bestCount, bestFreq := 0, -1
		for cnt, freq := range countFreq[k] {
			if freq > bestFreq || (freq == bestFreq && cnt < bestCount) {
				bestCount, bestFreq = cnt, freq
			}
		}
		c.Count = bestCount

		var bestPrint printID
		bestPrintFreq := -1
		for p, freq := range printFreq[k] {
			if freq > bestPrintFreq ||
				(freq == bestPrintFreq && (p.set < bestPrint.set || (p.set == bestPrint.set && p.number < bestPrint.number))) {
				bestPrint, bestPrintFreq = p, freq
			}
		}
		c.Set, c.Number = bestPrint.set, bestPrint.number

		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return CardKey(list[i]) < CardKey(list[j]) })
	return list
}

// coreHash hashes the sorted (cardKey, count-in-this-specific-deck) pairs
// restricted to core cards, so two decklists with the same skeleton but
// different tech/swap choices land on the same hash, and decklists that
// differ in how many copies of a *core* card they run land on different
// hashes (that's a real build difference, not just a tech swap). Copies
// that share a CardKey (Trainer/Energy reprints) are summed, so splitting the same
// total across different prints doesn't change the hash.
func coreHash(cards []models.Card, core map[string]bool) string {
	counts := map[string]int{}
	for _, c := range cards {
		k := CardKey(c)
		if core[k] {
			counts[k] += c.Count
		}
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s:%d;", k, counts[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}
