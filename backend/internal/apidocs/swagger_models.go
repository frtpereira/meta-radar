package apidocs

import (
	"time"

	"github.com/frtpereira/meta-radar/internal/models"
)

type ArchetypeStat struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	DeckCount   int      `json:"deck_count"`
	AvgStanding *float64 `json:"avg_standing,omitempty"`
	DropCount   int      `json:"drop_count"`
	Matches     int      `json:"matches"`
	Wins        int      `json:"wins"`
	Losses      int      `json:"losses"`
	Ties        int      `json:"ties"`
	ScoreRate   *float64 `json:"score_rate,omitempty"`
	WinRate     *float64 `json:"win_rate,omitempty"`
}

type ArchetypeDetail struct {
	ID             int64         `json:"id"`
	MetaID         string        `json:"meta_id"`
	Name           string        `json:"name"`
	Slug           string        `json:"slug"`
	CoreCards      []models.Card `json:"core_cards"`
	CoreThreshold  *float64      `json:"core_threshold,omitempty"`
	CoreComputedAt *time.Time    `json:"core_computed_at,omitempty"`
}

type Variant struct {
	CoreHash         string   `json:"core_hash"`
	DeckCount        int      `json:"deck_count"`
	AvgStanding      *float64 `json:"avg_standing,omitempty"`
	DropCount        int      `json:"drop_count"`
	SampleDecklistID int64    `json:"sample_decklist_id"`
}

type StandingRow struct {
	Standing      int     `json:"standing"`
	Wins          int     `json:"wins"`
	Losses        int     `json:"losses"`
	Ties          int     `json:"ties"`
	PlayerID      string  `json:"player_id"`
	PlayerName    string  `json:"player_name"`
	DecklistID    *int64  `json:"decklist_id,omitempty"`
	ArchetypeID   *int64  `json:"archetype_id,omitempty"`
	ArchetypeName *string `json:"archetype_name,omitempty"`
	ArchetypeSlug *string `json:"archetype_slug,omitempty"`
}

type TournamentDetail struct {
	ID            int64         `json:"id"`
	Name          string        `json:"name"`
	Game          string        `json:"game"`
	FormatCode    string        `json:"format_code"`
	MetaID        string        `json:"meta_id"`
	MetaName      string        `json:"meta_name"`
	Date          time.Time     `json:"date"`
	Players       int           `json:"players"`
	IsOnline      bool          `json:"is_online"`
	HasDecklists  bool          `json:"has_decklists"`
	OrganizerName string        `json:"organizer_name"`
	Standings     []StandingRow `json:"standings"`
}

type ArchetypeRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type MatchupStat struct {
	Archetype ArchetypeRef `json:"archetype"`
	Opponent  ArchetypeRef `json:"opponent"`
	Matches   int          `json:"matches"`
	Wins      int          `json:"wins"`
	Losses    int          `json:"losses"`
	Ties      int          `json:"ties"`
	ScoreRate *float64     `json:"score_rate,omitempty"`
	WinRate   *float64     `json:"win_rate,omitempty"`
}

type MatchupsResponse struct {
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
	PrevPage   int           `json:"prev_page"`
	NextPage   int           `json:"next_page"`
	PrevURL    string        `json:"prev_url,omitempty"`
	NextURL    string        `json:"next_url,omitempty"`
	Items      []MatchupStat `json:"items"`
}

type TournamentsResponse struct {
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
	PrevPage   int                 `json:"prev_page"`
	NextPage   int                 `json:"next_page"`
	PrevURL    string              `json:"prev_url,omitempty"`
	NextURL    string              `json:"next_url,omitempty"`
	Items      []models.Tournament `json:"items"`
}

type CardStat struct {
	Name              string             `json:"name"`
	Set               string             `json:"set"`
	Number            string             `json:"number"`
	Category          string             `json:"category"`
	IsCore            bool               `json:"is_core"`
	DeckCount         int                `json:"deck_count"`
	TotalDecklists    int                `json:"total_decklists"`
	Presence          float64            `json:"presence"`
	ModalCount        int                `json:"modal_count"`
	CountDistribution map[string]float64 `json:"count_distribution"`
}
