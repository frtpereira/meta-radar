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
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Game          string    `json:"game"`
	FormatCode    string    `json:"format_code"`
	MetaID        *string   `json:"meta_id,omitempty"`
	Date          time.Time `json:"date"`
	Players       int       `json:"players"`
	IsOnline      bool      `json:"is_online"`
	HasDecklists  bool      `json:"has_decklists"`
	OrganizerName *string   `json:"organizer_name,omitempty"`
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
	Standing     int    `json:"standing"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	Ties         int    `json:"ties"`
	DecklistID   *int64 `json:"decklist_id,omitempty"`
}
