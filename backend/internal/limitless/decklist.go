package limitless

import (
	"encoding/json"

	"github.com/frtpereira/meta-radar/internal/models"
)

// ParsePTCGDecklist attempts to normalize the raw, game-specific `decklist`
// field from a standings entry into our flat []models.Card shape.
//
// NOT YET VERIFIED against a live API response -- the docs mark this field
// as "format differs by game" without a schema. This tries the shape most
// consistent with Limitless's own deck builder export (cards grouped under
// "pokemon" / "trainer" / "energy" keys, each an array of
// {name, set, number, count}), and falls back to leaving `cards` empty
// while the raw bytes are always preserved separately in `raw_list` so
// nothing is lost if this guess is wrong.
//
// Before relying on this for anything real, fetch one live tournament's
// standings and check the actual shape of a non-null `decklist` field,
// then adjust this function accordingly.
func ParsePTCGDecklist(raw json.RawMessage) []models.Card {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var grouped struct {
		Pokemon []rawCard `json:"pokemon"`
		Trainer []rawCard `json:"trainer"`
		Energy  []rawCard `json:"energy"`
	}
	if err := json.Unmarshal(raw, &grouped); err == nil && (len(grouped.Pokemon)+len(grouped.Trainer)+len(grouped.Energy) > 0) {
		cards := make([]models.Card, 0, len(grouped.Pokemon)+len(grouped.Trainer)+len(grouped.Energy))
		cards = append(cards, toModelCards(grouped.Pokemon, "pokemon")...)
		cards = append(cards, toModelCards(grouped.Trainer, "trainer")...)
		cards = append(cards, toModelCards(grouped.Energy, "energy")...)
		return cards
	}

	// Fall back: maybe it's a flat array with a category field already on it.
	var flat []rawCard
	if err := json.Unmarshal(raw, &flat); err == nil && len(flat) > 0 {
		return toModelCards(flat, "")
	}

	return nil
}

type rawCard struct {
	Name     string `json:"name"`
	Set      string `json:"set"`
	Number   string `json:"number"`
	Count    int    `json:"count"`
	Category string `json:"category"`
}

func toModelCards(raw []rawCard, defaultCategory string) []models.Card {
	cards := make([]models.Card, 0, len(raw))
	for _, c := range raw {
		category := c.Category
		if category == "" {
			category = defaultCategory
		}
		cards = append(cards, models.Card{
			Name:     c.Name,
			Set:      c.Set,
			Number:   c.Number,
			Count:    c.Count,
			Category: category,
		})
	}
	return cards
}
