import type { ArchetypeStat, MatchupStat, Meta, Tournament } from "@/lib/types";

const apiBaseUrl =
    process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api";

async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(`${apiBaseUrl}${path}`, {
        cache: "no-store",
    });

    // Read response as text first so we can include server error bodies in thrown errors
    const text = await response.text();
    let parsed: any = null;
    try {
        parsed = text ? JSON.parse(text) : null;
    } catch (err) {
        // If parsing fails, keep the raw text in `parsed` so we can surface it
        parsed = text;
    }

    if (!response.ok) {
        const serverMessage =
            parsed && typeof parsed === "object" && "message" in parsed
                ? String((parsed as any).message)
                : typeof parsed === "string"
                ? parsed
                : undefined;

        throw new Error(
            `Request failed: ${response.status} ${response.statusText}${
                serverMessage ? ` - ${serverMessage}` : ""
            }",
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

export async function getMatchupStats(options: {
    metaId: string;
    archetypeId?: string;
    minMatches?: number;
    includeMirrors?: boolean;
}) {
    const params = new URLSearchParams();

    params.set("meta_id", options.metaId);
    params.set("min_matches", String(options.minMatches ?? 5));
    if (options.archetypeId) {
        params.set("archetype_id", options.archetypeId);
    }
    params.set("include_mirrors", options.includeMirrors ? "true" : "false");

    return fetchJson<MatchupStat[]>(`/matchups/stats?${params.toString()}`);
}
