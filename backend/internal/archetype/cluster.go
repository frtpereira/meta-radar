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

	"github.com/frtpereira/pokemon-tcg-tracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultCoreThreshold: a card played in at least 70% of an archetype's
// decklists counts as core. Below that, it's treated as a tech/swap slot.
const DefaultCoreThreshold = 0.7

type Clusterer struct {
	DB *pgxpool.Pool
}

func NewClusterer(db *pgxpool.Pool) *Clusterer {
	return &Clusterer{DB: db}
}

// cardKey canonically identifies a card for frequency counting: name + set
// + number. Two different prints of a functionally-identical card (e.g. a
// reprinted Boss's Orders from an older set) are treated as distinct here --
// a known simplification. Merging reprints would need a canonical-card
// mapping this project doesn't have yet; worth revisiting if it causes
// visibly fragmented cores in practice.
func cardKey(c models.Card) string {
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
			k := cardKey(c)
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

// coreCardList picks one representative Card per core key, using whichever
// count was most common for that card across decklists (so "core" reflects
// a realistic play count, not just presence).
func coreCardList(decks []deckRow, core map[string]bool) []models.Card {
	countFreq := map[string]map[int]int{} // cardKey -> count -> how many decklists played that count
	rep := map[string]models.Card{}
	for _, d := range decks {
		for _, c := range d.Cards {
			k := cardKey(c)
			if !core[k] {
				continue
			}
			if countFreq[k] == nil {
				countFreq[k] = map[int]int{}
			}
			countFreq[k][c.Count]++
			rep[k] = c
		}
	}

	list := make([]models.Card, 0, len(core))
	for k := range core {
		c := rep[k]
		bestCount, bestFreq := c.Count, -1
		for cnt, freq := range countFreq[k] {
			if freq > bestFreq {
				bestCount, bestFreq = cnt, freq
			}
		}
		c.Count = bestCount
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return cardKey(list[i]) < cardKey(list[j]) })
	return list
}

// coreHash hashes the sorted (cardKey, count-in-this-specific-deck) pairs
// restricted to core cards, so two decklists with the same skeleton but
// different tech/swap choices land on the same hash, and decklists that
// differ in how many copies of a *core* card they run land on different
// hashes (that's a real build difference, not just a tech swap).
func coreHash(cards []models.Card, core map[string]bool) string {
	type kv struct {
		key   string
		count int
	}
	var pairs []kv
	for _, c := range cards {
		k := cardKey(c)
		if core[k] {
			pairs = append(pairs, kv{k, c.Count})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })

	h := sha256.New()
	for _, p := range pairs {
		fmt.Fprintf(h, "%s:%d;", p.key, p.count)
	}
	return hex.EncodeToString(h.Sum(nil))
}
