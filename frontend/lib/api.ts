import type {
    ArchetypeDetail,
    ArchetypeStat,
    ArchetypeVariant,
    CardStat,
    DecklistDetail,
    MatchupStat,
    Meta,
    PlayerDetail,
    Tournament,
    TournamentDetail,
} from "@/lib/types";

const apiBaseUrl =
    process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api";

async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(`${apiBaseUrl}${path}`, {
        cache: "no-store",
    });

    // Read response as text first so we can include server error bodies in thrown errors
    const text = await response.text();
    let parsed: unknown = null;
    try {
        parsed = text ? JSON.parse(text) : null;
    } catch {
        // If parsing fails, keep the raw text in `parsed` so we can surface it
        parsed = text;
    }

    if (!response.ok) {
        let serverMessage: string | undefined;
        if (parsed && typeof parsed === "object") {
            const obj = parsed as Record<string, unknown>;
            if ("message" in obj) {
                const m = obj["message"];
                serverMessage = typeof m === "string" ? m : JSON.stringify(m);
            }
        } else if (typeof parsed === "string") {
            serverMessage = parsed;
        }

        throw new Error(
            `Request failed: ${response.status} ${response.statusText}${
                serverMessage ? ` - ${serverMessage}` : ""
            }`,
        );
    }

    return parsed as T;
}

export async function getMetas() {
    return fetchJson<Meta[]>("/metas");
}

export interface TournamentPage {
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
    prev_page: number;
    next_page: number;
    prev_url?: string;
    next_url?: string;
    items: Tournament[];
}

export async function getTournaments(options: {
    metaId?: string;
    minPlayers?: number;
    source?: "online" | "offline";
    dateFrom?: string;
    dateTo?: string;
    winnerArchetype?: string;
    eventName?: string;
    organizerName?: string;
    sortBy?: string;
    sortDir?: "asc" | "desc";
    page?: number;
    pageSize?: number;
}) {
    const params = new URLSearchParams();

    params.set("min_players", String(options.minPlayers ?? 32));
    if (options.metaId) {
        params.set("meta_id", options.metaId);
    }
    if (options.source) {
        params.set("source", options.source);
    }
    if (options.dateFrom) {
        params.set("date_from", options.dateFrom);
    }
    if (options.dateTo) {
        params.set("date_to", options.dateTo);
    }
    if (options.winnerArchetype) {
        params.set("winner_archetype", options.winnerArchetype);
    }
    if (options.eventName) {
        params.set("event_name", options.eventName);
    }
    if (options.organizerName) {
        params.set("organizer_name", options.organizerName);
    }
    if (options.sortBy) {
        params.set("sort_by", options.sortBy);
        params.set("sort_dir", options.sortDir ?? "asc");
    }
    params.set("page", String(options.page ?? 1));
    params.set("page_size", String(options.pageSize ?? 20));

    return fetchJson<TournamentPage>(`/tournaments?${params.toString()}`);
}

export async function getTournament(id: string) {
    return fetchJson<TournamentDetail>(`/tournaments/${id}`);
}

export async function getArchetypeStats(metaId: string) {
    return fetchJson<ArchetypeStat[]>(`/archetypes/stats?meta_id=${metaId}`);
}

export async function getArchetypeDetail(id: string) {
    return fetchJson<ArchetypeDetail>(`/archetypes/${id}`);
}

export async function getArchetypeVariants(id: string) {
    return fetchJson<ArchetypeVariant[]>(`/archetypes/${id}/variants`);
}

export async function getArchetypeCardStats(id: string) {
    return fetchJson<CardStat[]>(`/archetypes/${id}/card-stats`);
}

export async function getPlayer(nickname: string) {
    return fetchJson<PlayerDetail>(
        `/players/${encodeURIComponent(nickname)}`,
    );
}

export async function getDecklist(id: string) {
    return fetchJson<DecklistDetail>(`/decklists/${id}`);
}

export async function getMatchupStats(options: {
    metaId: string;
    archetypeId?: string;
    minMatches?: number;
    includeMirrors?: boolean;
}) {
    const params = new URLSearchParams();

    params.set("meta_id", options.metaId);
    params.set("min_matches", String(options.minMatches ?? 20));
    if (options.archetypeId) {
        params.set("archetype_id", options.archetypeId);
    }
    if (options.includeMirrors !== undefined) {
        params.set(
            "include_mirrors",
            options.includeMirrors ? "true" : "false",
        );
    }

    // Not paginated server-side: matchups_mv rows for one meta are bounded
    // by archetype-pair count, small enough to sort/paginate client-side
    // (see MatchupTable) without a refetch per sort/page change.
    return fetchJson<MatchupStat[]>(`/matchups/stats?${params.toString()}`);
}
