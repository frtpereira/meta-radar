export interface Meta {
    id: string;
    name: string;
    format_code: string;
    starts_at: string;
    ends_at: string | null;
}

export interface Tournament {
    id: string;
    name: string;
    game: string;
    format_code: string;
    meta_id: string | null;
    date: string;
    players: number;
    is_online: boolean;
    has_decklists: boolean;
    organizer_name: string | null;
    winner_archetype: string | null;
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
}

export interface TournamentDetail extends Tournament {
    standings: TournamentStanding[];
}

interface MatchupDeck {
    id: number;
    name: string;
    slug: string;
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
