import { afterEach, describe, expect, it, vi } from "vitest";
import {
    getArchetypeCardStats,
    getArchetypeDetail,
    getArchetypeStats,
    getArchetypeVariants,
    getMatchupStats,
    getMetas,
    getTournament,
    getTournaments,
} from "./api";

const API_BASE = "http://localhost:8080/api";

type FetchResponseOptions = {
    ok: boolean;
    status: number;
    statusText: string;
    body?: string;
};

function mockFetchResponse({ ok, status, statusText, body = "" }: FetchResponseOptions) {
    const fetchMock = vi.fn().mockResolvedValue({
        ok,
        status,
        statusText,
        text: vi.fn().mockResolvedValue(body),
    });

    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
}

const successCases = [
    {
        name: "getMetas",
        call: () => getMetas(),
        expectedPath: "/metas",
        payload: [
            {
                id: "meta-1",
                name: "Worlds 2026",
                format_code: "STD",
                starts_at: "2026-01-01",
                ends_at: null,
            },
        ],
    },
    {
        name: "getTournaments",
        call: () => getTournaments({}),
        expectedPath: "/tournaments?min_players=32&page=1&page_size=20",
        payload: {
            total: 1,
            page: 1,
            page_size: 20,
            total_pages: 1,
            prev_page: 0,
            next_page: 0,
            items: [
                {
                    id: "tour-1",
                    name: "League Cup",
                    game: "pokemon",
                    format_code: "STD",
                    meta_id: "meta-1",
                    meta_name: "Worlds 2026",
                    date: "2026-02-03",
                    players: 64,
                    is_online: false,
                    has_decklists: true,
                    organizer_name: "TO",
                    winner_archetype: "Charizard",
                },
            ],
        },
    },
    {
        name: "getTournament",
        call: () => getTournament("tour-1"),
        expectedPath: "/tournaments/tour-1",
        payload: {
            id: "tour-1",
            name: "League Cup",
            game: "pokemon",
            format_code: "STD",
            meta_id: "meta-1",
            meta_name: "Worlds 2026",
            date: "2026-02-03",
            players: 64,
            is_online: false,
            has_decklists: true,
            organizer_name: "TO",
            winner_archetype: "Charizard",
            standings: [],
        },
    },
    {
        name: "getArchetypeStats",
        call: () => getArchetypeStats("meta-1"),
        expectedPath: "/archetypes/stats?meta_id=meta-1",
        payload: [
            {
                id: 1,
                name: "Charizard",
                slug: "charizard",
                deck_count: 12,
                avg_standing: 5,
                drop_count: 0,
                matches: 30,
                wins: 18,
                losses: 10,
                ties: 2,
                score_rate: 0.63,
                win_rate: 0.64,
            },
        ],
    },
    {
        name: "getArchetypeDetail",
        call: () => getArchetypeDetail("5"),
        expectedPath: "/archetypes/5",
        payload: {
            id: 5,
            meta_id: "meta-1",
            name: "Gardevoir",
            slug: "gardevoir",
            core_cards: null,
            core_threshold: null,
            core_computed_at: null,
        },
    },
    {
        name: "getArchetypeVariants",
        call: () => getArchetypeVariants("5"),
        expectedPath: "/archetypes/5/variants",
        payload: [
            {
                core_hash: "abc123",
                deck_count: 4,
                avg_standing: 8,
                drop_count: 0,
                sample_decklist_id: 44,
            },
        ],
    },
    {
        name: "getArchetypeCardStats",
        call: () => getArchetypeCardStats("5"),
        expectedPath: "/archetypes/5/card-stats",
        payload: [
            {
                name: "Rare Candy",
                set: "SVI",
                number: "191",
                category: "trainer",
                is_core: true,
                deck_count: 10,
                total_decklists: 12,
                presence: 0.83,
                modal_count: 4,
                count_distribution: { "4": 0.83 },
            },
        ],
    },
    {
        name: "getMatchupStats",
        call: () => getMatchupStats({ metaId: "meta-1" }),
        expectedPath:
            "/matchups/stats?meta_id=meta-1&min_matches=20&page=1&page_size=20",
        payload: {
            total: 1,
            page: 1,
            page_size: 20,
            total_pages: 1,
            prev_page: 0,
            next_page: 0,
            items: [
                {
                    archetype: { id: 1, name: "Charizard", slug: "charizard" },
                    opponent: { id: 2, name: "Gardevoir", slug: "gardevoir" },
                    matches: 18,
                    wins: 10,
                    losses: 6,
                    ties: 2,
                    score_rate: 0.61,
                    win_rate: 0.63,
                },
            ],
        },
    },
] as const;

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("api fetch helpers", () => {
    describe.each(successCases)("$name", ({ call, expectedPath, payload }) => {
        it("returns parsed JSON for successful responses", async () => {
            const fetchMock = mockFetchResponse({
                ok: true,
                status: 200,
                statusText: "OK",
                body: JSON.stringify(payload),
            });

            await expect(call()).resolves.toEqual(payload);
            expect(fetchMock).toHaveBeenCalledWith(`${API_BASE}${expectedPath}`, {
                cache: "no-store",
            });
        });

        it("returns null for successful empty responses", async () => {
            mockFetchResponse({
                ok: true,
                status: 204,
                statusText: "No Content",
                body: "",
            });

            await expect(call()).resolves.toBeNull();
        });

        it("extracts JSON error messages from non-ok responses", async () => {
            mockFetchResponse({
                ok: false,
                status: 400,
                statusText: "Bad Request",
                body: JSON.stringify({ message: "Bad filters" }),
            });

            await expect(call()).rejects.toThrow(
                "Request failed: 400 Bad Request - Bad filters",
            );
        });

        it("extracts plain-text error messages from non-ok responses", async () => {
            mockFetchResponse({
                ok: false,
                status: 500,
                statusText: "Server Error",
                body: "Backend unavailable",
            });

            await expect(call()).rejects.toThrow(
                "Request failed: 500 Server Error - Backend unavailable",
            );
        });

        it.each([
            ["{not-json", "Request failed: 502 Bad Gateway - {not-json"],
            ["", "Request failed: 502 Bad Gateway"],
        ])(
            "handles non-ok responses with body %p",
            async (body, expectedMessage) => {
                mockFetchResponse({
                    ok: false,
                    status: 502,
                    statusText: "Bad Gateway",
                    body,
                });

                await expect(call()).rejects.toThrow(expectedMessage);
            },
        );
    });

    it("builds the default tournaments query string", async () => {
        const fetchMock = mockFetchResponse({
            ok: true,
            status: 200,
            statusText: "OK",
            body: JSON.stringify({ items: [] }),
        });

        await getTournaments({});

        expect(fetchMock).toHaveBeenCalledWith(
            `${API_BASE}/tournaments?min_players=32&page=1&page_size=20`,
            { cache: "no-store" },
        );
    });

    it("builds the tournaments query string with every optional filter set", async () => {
        const fetchMock = mockFetchResponse({
            ok: true,
            status: 200,
            statusText: "OK",
            body: JSON.stringify({ items: [] }),
        });

        await getTournaments({
            metaId: "meta-1",
            minPlayers: 64,
            source: "online",
            dateFrom: "2026-03-01",
            dateTo: "2026-03-31",
            winnerArchetype: "charizard",
            page: 3,
            pageSize: 50,
        });

        expect(fetchMock).toHaveBeenCalledWith(
            `${API_BASE}/tournaments?min_players=64&meta_id=meta-1&source=online&date_from=2026-03-01&date_to=2026-03-31&winner_archetype=charizard&page=3&page_size=50`,
            { cache: "no-store" },
        );
    });

    it("builds the default matchup query string without optional params", async () => {
        const fetchMock = mockFetchResponse({
            ok: true,
            status: 200,
            statusText: "OK",
            body: JSON.stringify({ items: [] }),
        });

        await getMatchupStats({ metaId: "meta-1" });

        expect(fetchMock).toHaveBeenCalledWith(
            `${API_BASE}/matchups/stats?meta_id=meta-1&min_matches=20&page=1&page_size=20`,
            { cache: "no-store" },
        );
    });

    it("builds the matchup query string with every optional filter set", async () => {
        const fetchMock = mockFetchResponse({
            ok: true,
            status: 200,
            statusText: "OK",
            body: JSON.stringify({ items: [] }),
        });

        await getMatchupStats({
            metaId: "meta-1",
            archetypeId: "7",
            minMatches: 5,
            includeMirrors: false,
            page: 2,
            pageSize: 40,
        });

        expect(fetchMock).toHaveBeenCalledWith(
            `${API_BASE}/matchups/stats?meta_id=meta-1&min_matches=5&archetype_id=7&include_mirrors=false&page=2&page_size=40`,
            { cache: "no-store" },
        );
    });
});
