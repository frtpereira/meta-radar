import type { ArchetypeStat, MatchupStat, Meta, Tournament } from "@/lib/types";

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
            }`
        );
    }

    return parsed as T;
}

export async function getMetas() {
    return fetchJson<Meta[]>("/metas");
}

export async function getTournaments(options: {
    metaId?: string;
    minPlayers?: number;
}) {
    const params = new URLSearchParams();

    params.set("min_players", String(options.minPlayers ?? 64));
    if (options.metaId) {
        params.set("meta_id", options.metaId);
    }

    return fetchJson<Tournament[]>(`/tournaments?${params.toString()}`);
}

export async function getArchetypeStats(metaId: string) {
    return fetchJson<ArchetypeStat[]>(`/archetypes/stats?meta_id=${metaId}`);
}

export interface MatchupPage {
    total: number;
    page: number;
    page_size: number;
    items: MatchupStat[];
}

export async function getMatchupStats(options: {
    metaId: string;
    archetypeId?: string;
    minMatches?: number;
    includeMirrors?: boolean;
    page?: number;
    pageSize?: number;
}) {
    const params = new URLSearchParams();

    params.set("meta_id", options.metaId);
    params.set("min_matches", String(options.minMatches ?? 20));
    if (options.archetypeId) {
        params.set("archetype_id", options.archetypeId);
    }
    params.set("include_mirrors", options.includeMirrors ? "true" : "false");
    params.set("page", String(options.page ?? 1));
    params.set("page_size", String(options.pageSize ?? 20));

    return fetchJson<MatchupPage>(`/matchups/stats?${params.toString()}`);
}
