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
