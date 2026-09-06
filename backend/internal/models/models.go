package models

import "time"

type Meta struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	FormatCode string     `json:"format_code"`
	StartsAt   time.Time  `json:"starts_at"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
}

type Tournament struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Game            string    `json:"game"`
	FormatCode      string    `json:"format_code"`
	MetaID          *string   `json:"meta_id,omitempty"`
	MetaName        *string   `json:"meta_name,omitempty"`
	Date            time.Time `json:"date"`
	Players         int       `json:"players"`
	IsOnline        bool      `json:"is_online"`
	HasDecklists    bool      `json:"has_decklists"`
	OrganizerName   *string   `json:"organizer_name,omitempty"`
	WinnerArchetype *string   `json:"winner_archetype,omitempty"`
	// WinnerArchetypeIcons holds the ordered pokemon-icon slugs for the
	// winner's archetype (see archetype_icons table), so the frontend can
	// render icons instead of the archetype name. Nil when the winner has
	// no archetype or no curated icons.
	WinnerArchetypeIcons []string `json:"winner_archetype_icons,omitempty"`
}

type Archetype struct {
	ID     int64  `json:"id"`
	MetaID string `json:"meta_id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
}

type Card struct {
	Name     string `json:"name"`
	Set      string `json:"set,omitempty"`
	Number   string `json:"number,omitempty"`
	Count    int    `json:"count"`
	Category string `json:"category"` // "pokemon" | "trainer" | "energy"
}

type Decklist struct {
	ID           int64  `json:"id"`
	TournamentID string `json:"tournament_id"`
	PlayerID     string `json:"player_id"`
	ArchetypeID  *int64 `json:"archetype_id,omitempty"`
	Cards        []Card `json:"cards"`
	CoreHash     string `json:"core_hash,omitempty"`
}

type Standing struct {
	TournamentID string `json:"tournament_id"`
	PlayerID     string `json:"player_id"`
	Standing     int    `json:"standing"` // final rank; 0 means the player dropped
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	Ties         int    `json:"ties"`
	DecklistID   *int64 `json:"decklist_id,omitempty"`
}

// PokemonIcon is one entry in our self-hosted icon registry (see
// db/migrations/0006_pokemon_icons.sql and fetch_pokemon_icons.py). Slug
// matches the icon's base filename in our R2 bucket.
type PokemonIcon struct {
	Slug       string    `json:"slug"`
	R2Key      string    `json:"r2_key"`
	Gen        *int      `json:"gen,omitempty"`
	ResolvedAt time.Time `json:"resolved_at"`
}

// CardPokemonIcon maps a card's exact printed name to the Pokémon icon
// slug it should render as. Curated, not derived -- see the migration
// comment for why a plain string transform isn't reliable here.
type CardPokemonIcon struct {
	CardName    string `json:"card_name"`
	PokemonSlug string `json:"pokemon_slug"`
}

// ArchetypeIcon links an archetype to one of the icon(s) that represent
// it, in display order. An archetype can have more than one (e.g.
// dual-attacker builds), so this is a separate row per icon rather than
// a column on Archetype.
type ArchetypeIcon struct {
	ArchetypeID  int64  `json:"archetype_id"`
	PokemonSlug  string `json:"pokemon_slug"`
	DisplayOrder int    `json:"display_order"`
}
