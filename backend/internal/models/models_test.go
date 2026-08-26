package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelJSONRoundTrips(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	metaID := "meta-1"
	metaName := "Standard 2026"
	organizer := "League Cup"
	winner := "Dragapult ex"
	archetypeID := int64(42)
	decklistID := int64(99)

	tests := []struct {
		name  string
		value any
		newFn func() any
	}{
		{
			name:  "meta",
			value: Meta{ID: metaID, Name: metaName, FormatCode: "STANDARD", StartsAt: start, EndsAt: &end},
			newFn: func() any { return &Meta{} },
		},
		{
			name: "tournament",
			value: Tournament{
				ID:              "t1",
				Name:            "Regional",
				Game:            "PTCG",
				FormatCode:      "STANDARD",
				MetaID:          &metaID,
				MetaName:        &metaName,
				Date:            start,
				Players:         256,
				IsOnline:        true,
				HasDecklists:    true,
				OrganizerName:   &organizer,
				WinnerArchetype: &winner,
			},
			newFn: func() any { return &Tournament{} },
		},
		{
			name:  "archetype",
			value: Archetype{ID: 5, MetaID: metaID, Name: "Dragapult ex", Slug: "dragapult-ex"},
			newFn: func() any { return &Archetype{} },
		},
		{
			name:  "card",
			value: Card{Name: "Buddy-Buddy Poffin", Set: "TEF", Number: "144", Count: 4, Category: "trainer"},
			newFn: func() any { return &Card{} },
		},
		{
			name: "decklist",
			value: Decklist{
				ID:           decklistID,
				TournamentID: "t1",
				PlayerID:     "p1",
				ArchetypeID:  &archetypeID,
				Cards:        []Card{{Name: "Rare Candy", Count: 4, Category: "trainer"}},
				CoreHash:     "abc123",
			},
			newFn: func() any { return &Decklist{} },
		},
		{
			name: "standing",
			value: Standing{
				TournamentID: "t1",
				PlayerID:     "p1",
				Standing:     1,
				Wins:         9,
				Losses:       1,
				Ties:         0,
				DecklistID:   &decklistID,
			},
			newFn: func() any { return &Standing{} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)

			got := tt.newFn()
			require.NoError(t, json.Unmarshal(data, got))
			assert.Equal(t, tt.value, deref(got))
		})
	}
}

func TestModelJSONOmitemptyBehavior(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	metaData, err := json.Marshal(Meta{ID: "meta", Name: "Meta", FormatCode: "STD", StartsAt: start})
	require.NoError(t, err)
	assert.NotContains(t, string(metaData), "ends_at")

	tournamentData, err := json.Marshal(Tournament{ID: "t1", Name: "Cup", Game: "PTCG", FormatCode: "STD", Date: start})
	require.NoError(t, err)
	assert.NotContains(t, string(tournamentData), "meta_id")
	assert.NotContains(t, string(tournamentData), "meta_name")
	assert.NotContains(t, string(tournamentData), "organizer_name")
	assert.NotContains(t, string(tournamentData), "winner_archetype")

	cardData, err := json.Marshal(Card{Name: "Switch", Count: 2, Category: "trainer"})
	require.NoError(t, err)
	assert.NotContains(t, string(cardData), "set")
	assert.NotContains(t, string(cardData), "number")

	decklistData, err := json.Marshal(Decklist{ID: 1, TournamentID: "t1", PlayerID: "p1", Cards: []Card{}})
	require.NoError(t, err)
	assert.NotContains(t, string(decklistData), "archetype_id")
	assert.NotContains(t, string(decklistData), "core_hash")

	standingData, err := json.Marshal(Standing{TournamentID: "t1", PlayerID: "p1"})
	require.NoError(t, err)
	assert.NotContains(t, string(standingData), "decklist_id")
}

func deref(v any) any {
	switch value := v.(type) {
	case *Meta:
		return *value
	case *Tournament:
		return *value
	case *Archetype:
		return *value
	case *Card:
		return *value
	case *Decklist:
		return *value
	case *Standing:
		return *value
	default:
		return v
	}
}
