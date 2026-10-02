export type MetaType = "standard" | "set";

export interface Meta {
    id: string;
    name: string;
    format_code: string;
    type: MetaType;
    // These two are pointer fields on the Go side with `omitempty`, so
    // an open meta's `ends_at` and a standard meta's `parent_meta_id`
    // aren't sent as `null` -- the key is dropped from the response
    // entirely (see backend/internal/models/models_test.go's
    // TestModelJSONOmitemptyBehavior). Treat both as "not set" with a
    // falsy check (`!m.ends_at`), never `=== null` -- `undefined` fails
    // that check silently instead of erroring, which is exactly what
    // broke default-meta selection before this comment existed.
    parent_meta_id?: string | null;
    starts_at: string;
    ends_at?: string | null;
}

export interface CurrentMetas {
    standard: Meta | null;
    current_set: Meta | null;
}

export interface Tournament {
    id: string;
    name: string;
    game: string;
    format_code: string;
    meta_id: string | null;
    meta_name: string | null;
    date: string;
    players: number;
    is_online: boolean;
    ingest_source: "play_api" | "labs";
    has_decklists: boolean;
    organizer_name: string | null;
    winner_archetype: string | null;
    winner_archetype_icons: string[] | null;
    winner_nickname: string | null;
    winner_decklist_id: number | null;
    is_current_standard: boolean;
}

export interface ArchetypeStat {
    id: number;
    name: string;
    slug: string;
    deck_count: number;
    avg_standing: number | null;
    drop_count: number;
    matches: number;
    wins: number;
    losses: number;
    ties: number;
    score_rate: number | null;
    win_rate: number | null;
    archetype_icons: string[] | null;
}

export interface TournamentStanding {
    standing: number;
    wins: number;
    losses: number;
    ties: number;
    player_id: string;
    player_name: string;
    decklist_id: number | null;
    archetype_id: number | null;
    archetype_name: string | null;
    archetype_slug: string | null;
    archetype_icons: string[] | null;
}

export interface TournamentDivision {
    id: string;
    division: "MA" | "SR" | "JR";
}

export interface TournamentDetail extends Tournament {
    is_official: boolean;
    division: "MA" | "SR" | "JR" | null;
    divisions: TournamentDivision[];
    standings: TournamentStanding[];
}

interface MatchupDeck {
    id: number;
    name: string;
    slug: string;
    icons: string[] | null;
}

export interface Card {
    name: string;
    set: string;
    number: string;
    count: number;
    category: string; // "pokemon" | "trainer" | "energy"
}

export interface ArchetypeDetail {
    id: number;
    meta_id: string;
    name: string;
    slug: string;
    core_cards: Card[] | null;
    core_threshold: number | null;
    core_computed_at: string | null;
    archetype_icons: string[] | null;
}

export interface ArchetypeVariant {
    core_hash: string;
    deck_count: number;
    avg_standing: number | null;
    drop_count: number;
    sample_decklist_id: number;
}

export interface CardStat {
    name: string;
    set: string;
    number: string;
    category: string;
    is_core: boolean;
    deck_count: number;
    total_decklists: number;
    presence: number;
    modal_count: number;
    count_distribution: Record<string, number>; // copy count string -> fraction of all decklists
}

export interface MatchupStat {
    archetype: MatchupDeck;
    opponent: MatchupDeck;
    matches: number;
    wins: number;
    losses: number;
    ties: number;
    score_rate: number | null;
    win_rate: number | null;
}

export interface PlayerHistoryEntry {
    tournament_id: string;
    event_name: string;
    date: string;
    players: number;
    placement: number;
    decklist_id: number | null;
    archetype_id: number | null;
    archetype_name: string | null;
    archetype_slug: string | null;
    archetype_icons: string[] | null;
}

export interface PlayerDetail {
    id: string;
    name: string;
    history: PlayerHistoryEntry[];
}

export type PairingOutcome = "win" | "loss" | "draw" | "bye" | "unknown";

export interface PairingRow {
    phase: number;
    round: number;
    table_number: number;
    outcome: PairingOutcome;
    opponent_id: string | null;
    opponent_name: string | null;
    opponent_decklist_id: number | null;
    opponent_archetype_id: number | null;
    opponent_archetype_name: string | null;
    opponent_archetype_slug: string | null;
    opponent_archetype_icons: string[] | null;
}

export interface PairingsDetail {
    tournament_id: string;
    tournament_name: string;
    player_id: string;
    player_name: string;
    pairings: PairingRow[];
}

export interface DecklistDetail {
    id: number;
    tournament_id: string;
    tournament_name: string;
    date: string;
    player_id: string;
    player_name: string;
    archetype_id: number | null;
    archetype_name: string | null;
    archetype_slug: string | null;
    archetype_icons: string[] | null;
    cards: Card[];
}

export interface MatchupCardRecommendation {
    name: string;
    category: string;
    matches_with: number;
    score_rate_with: number;
    matches_without: number;
    score_rate_without: number;
    delta: number;
    recommendation: "include" | "cut" | "neutral";
}

export interface MatchupCards {
    archetype_id: string;
    opponent_id: string;
    matches: number;
    score_rate: number | null;
    recommendations: MatchupCardRecommendation[];
}
