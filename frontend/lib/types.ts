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
