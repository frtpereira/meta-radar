package limitless

import (
	"encoding/json"
	"testing"

	"github.com/frtpereira/meta-radar/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestParsePTCGDecklist(t *testing.T) {
	t.Run("grouped cards", func(t *testing.T) {
		raw := json.RawMessage(`{"pokemon":[{"name":"Drakloak","set":"TWM","number":"129","count":4}],"trainer":[{"name":"Buddy-Buddy Poffin","set":"TEF","number":"144","count":3}],"energy":[{"name":"Psychic Energy","count":6}]}`)
		assert.Equal(t, []models.Card{
			{Name: "Drakloak", Set: "TWM", Number: "129", Count: 4, Category: "pokemon"},
			{Name: "Buddy-Buddy Poffin", Set: "TEF", Number: "144", Count: 3, Category: "trainer"},
			{Name: "Psychic Energy", Count: 6, Category: "energy"},
		}, ParsePTCGDecklist(raw))
	})

	t.Run("flat cards", func(t *testing.T) {
		raw := json.RawMessage(`[{"name":"Switch","set":"SVI","number":"194","count":2,"category":"trainer"},{"name":"Basic Water Energy","count":8}]`)
		assert.Equal(t, []models.Card{
			{Name: "Switch", Set: "SVI", Number: "194", Count: 2, Category: "trainer"},
			{Name: "Basic Water Energy", Count: 8, Category: ""},
		}, ParsePTCGDecklist(raw))
	})

	for name, raw := range map[string]json.RawMessage{
		"nil":       nil,
		"empty":     json.RawMessage{},
		"null":      json.RawMessage("null"),
		"malformed": json.RawMessage(`{"pokemon":`),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, ParsePTCGDecklist(raw))
		})
	}
}

func TestToModelCardsUsesDefaultCategory(t *testing.T) {
	cards := toModelCards([]rawCard{
		{Name: "Rare Candy", Set: "SVI", Number: "191", Count: 4},
		{Name: "Boss's Orders", Set: "PAL", Number: "172", Count: 2, Category: "supporter"},
	}, "trainer")

	assert.Equal(t, []models.Card{
		{Name: "Rare Candy", Set: "SVI", Number: "191", Count: 4, Category: "trainer"},
		{Name: "Boss's Orders", Set: "PAL", Number: "172", Count: 2, Category: "supporter"},
	}, cards)
}
