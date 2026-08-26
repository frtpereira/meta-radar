package archetype

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/frtpereira/meta-radar/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCardKey(t *testing.T) {
	assert.Equal(t, "Buddy-Buddy Poffin|TEF|144", cardKey(models.Card{Name: "Buddy-Buddy Poffin", Set: "TEF", Number: "144"}))
}

func TestCoreCardListSelectsDeterministicCountsAndOrdering(t *testing.T) {
	decks := []deckRow{
		{ID: 1, Cards: []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 4, Category: "pokemon"}, {Name: "B", Set: "SET", Number: "2", Count: 2, Category: "trainer"}}},
		{ID: 2, Cards: []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 3, Category: "pokemon"}, {Name: "B", Set: "SET", Number: "2", Count: 2, Category: "trainer"}}},
	}
	core := map[string]bool{"A|SET|1": true, "B|SET|2": true}

	assert.Equal(t, []models.Card{
		{Name: "A", Set: "SET", Number: "1", Count: 3, Category: "pokemon"},
		{Name: "B", Set: "SET", Number: "2", Count: 2, Category: "trainer"},
	}, coreCardList(decks, core))

	assert.Empty(t, coreCardList(nil, map[string]bool{}))
}

func TestCoreHashUsesOnlyCoreCardsAndIsOrderIndependent(t *testing.T) {
	core := map[string]bool{"A|SET|1": true, "B|SET|2": true}
	cards1 := []models.Card{{Name: "B", Set: "SET", Number: "2", Count: 2}, {Name: "A", Set: "SET", Number: "1", Count: 4}, {Name: "Tech", Set: "SET", Number: "9", Count: 1}}
	cards2 := []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 4}, {Name: "B", Set: "SET", Number: "2", Count: 2}}

	assert.Equal(t, coreHash(cards1, core), coreHash(cards2, core))
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", coreHash(nil, map[string]bool{}))
}

func TestRunForMeta(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM archetypes WHERE meta_id = $1`)).
			WithArgs("meta-1").
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cards FROM decklists WHERE archetype_id = $1`)).WithArgs(int64(1)).WillReturnRows(pgxmock.NewRows([]string{"id", "cards"}))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cards FROM decklists WHERE archetype_id = $1`)).WithArgs(int64(2)).WillReturnRows(pgxmock.NewRows([]string{"id", "cards"}))

		cl := NewClusterer(mock)
		require.NoError(t, cl.RunForMeta(context.Background(), "meta-1", DefaultCoreThreshold))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM archetypes WHERE meta_id = $1`)).WithArgs("meta-1").WillReturnError(errors.New("db down"))

		cl := NewClusterer(mock)
		err = cl.RunForMeta(context.Background(), "meta-1", DefaultCoreThreshold)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing archetypes")
	})

	t.Run("scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM archetypes WHERE meta_id = $1`)).
			WithArgs("meta-1").
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("bad-id"))

		cl := NewClusterer(mock)
		err = cl.RunForMeta(context.Background(), "meta-1", DefaultCoreThreshold)
		require.Error(t, err)
	})
}

func TestRunForArchetype(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		deck1 := mustJSON(t, []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 4, Category: "pokemon"}, {Name: "B", Set: "SET", Number: "2", Count: 2, Category: "trainer"}})
		deck2 := mustJSON(t, []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 4, Category: "pokemon"}, {Name: "Tech", Set: "SET", Number: "9", Count: 1, Category: "trainer"}})
		deck3 := mustJSON(t, []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 3, Category: "pokemon"}, {Name: "B", Set: "SET", Number: "2", Count: 2, Category: "trainer"}})
		expectedCore := []models.Card{{Name: "A", Set: "SET", Number: "1", Count: 4, Category: "pokemon"}, {Name: "B", Set: "SET", Number: "2", Count: 2, Category: "trainer"}}
		threshold := 2.0 / 3.0

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cards FROM decklists WHERE archetype_id = $1`)).
			WithArgs(int64(7)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "cards"}).AddRow(int64(11), deck1).AddRow(int64(12), deck2).AddRow(int64(13), deck3))
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE archetypes SET core_cards = $1, core_threshold = $2, core_computed_at = now()
		WHERE id = $3`)).
			WithArgs(jsonArg(expectedCore), threshold, int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE decklists SET core_hash = $1 WHERE id = $2`)).WithArgs(coreHash(mustCards(t, deck1), map[string]bool{"A|SET|1": true, "B|SET|2": true}), int64(11)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE decklists SET core_hash = $1 WHERE id = $2`)).WithArgs(coreHash(mustCards(t, deck2), map[string]bool{"A|SET|1": true, "B|SET|2": true}), int64(12)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE decklists SET core_hash = $1 WHERE id = $2`)).WithArgs(coreHash(mustCards(t, deck3), map[string]bool{"A|SET|1": true, "B|SET|2": true}), int64(13)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		cl := NewClusterer(mock)
		require.NoError(t, cl.RunForArchetype(context.Background(), 7, threshold))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid deck json", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cards FROM decklists WHERE archetype_id = $1`)).
			WithArgs(int64(7)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "cards"}).AddRow(int64(11), []byte(`{"cards":`)))

		cl := NewClusterer(mock)
		err = cl.RunForArchetype(context.Background(), 7, DefaultCoreThreshold)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding cards")
	})

	t.Run("begin error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cards FROM decklists WHERE archetype_id = $1`)).
			WithArgs(int64(7)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "cards"}).AddRow(int64(11), mustJSON(t, []models.Card{{Name: "A", Count: 4, Category: "pokemon"}})))
		mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

		cl := NewClusterer(mock)
		err = cl.RunForArchetype(context.Background(), 7, DefaultCoreThreshold)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "begin failed")
	})

	t.Run("empty decklists", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cards FROM decklists WHERE archetype_id = $1`)).
			WithArgs(int64(7)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "cards"}))

		cl := NewClusterer(mock)
		require.NoError(t, cl.RunForArchetype(context.Background(), 7, DefaultCoreThreshold))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func mustCards(t *testing.T, raw []byte) []models.Card {
	t.Helper()
	var cards []models.Card
	require.NoError(t, json.Unmarshal(raw, &cards))
	return cards
}

type jsonArg []models.Card

func (j jsonArg) Match(v interface{}) bool {
	raw, ok := v.([]byte)
	if !ok {
		return false
	}
	var got []models.Card
	if err := json.Unmarshal(raw, &got); err != nil {
		return false
	}
	return assert.ObjectsAreEqual([]models.Card(j), got)
}

var _ = pgconn.CommandTag{}
