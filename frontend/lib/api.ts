import type { ArchetypeStat, Meta, Tournament } from "@/lib/types";

const apiBaseUrl =
    process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api";

async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(`${apiBaseUrl}${path}`, {
        cache: "no-store",
    });

    if (!response.ok) {
        throw new Error(
            `Request failed: ${response.status} ${response.statusText}`,
        );
    }

    return response.json() as Promise<T>;
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
