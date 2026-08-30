import http from "node:http";

const host = "127.0.0.1";
const port = 4100;

const metas = [
    {
        id: "meta-2026",
        name: "Worlds 2026",
        format_code: "STD",
        starts_at: "2026-05-01",
        ends_at: null,
    },
    {
        id: "meta-empty",
        name: "Testing Empty Meta",
        format_code: "STD",
        starts_at: "2026-06-01",
        ends_at: null,
    },
    {
        id: "meta-error",
        name: "Testing Error Meta",
        format_code: "STD",
        starts_at: "2026-07-01",
        ends_at: null,
    },
];

const archetypeNames = [
    "Charizard ex",
    "Gardevoir ex",
    "Dragapult ex",
    "Raging Bolt ex",
    "Lost Box",
    "Miraidon ex",
    "Palkia VSTAR",
    "Terapagos ex",
    "Roaring Moon ex",
    "Hydreigon ex",
    "Future Hands",
    "Regidrago VSTAR",
    "Lugia VSTAR",
    "Banette / Froslass",
    "Snorlax Stall",
    "Ancient Box",
    "Klawf",
    "Ogerpon Control",
    "Tinkaton ex",
    "Chien-Pao ex",
    "Iron Thorns ex",
    "Greninja ex",
    "Ceruledge ex",
    "Lokix",
    "Late Game Dragon",
];

const archetypeStats = archetypeNames.map((name, index) => ({
    id: index + 1,
    name,
    slug: slugify(name),
    deck_count: 60 - index,
    avg_standing: 4 + index / 3,
    drop_count: index % 3,
    matches: 40 + index * 3,
    wins: 22 + index,
    losses: 12 + (index % 5),
    ties: index % 2,
    score_rate: 0.58 - index * 0.004,
    win_rate: 0.6 - index * 0.004,
}));

const archetypeById = new Map(
    archetypeStats.map((stat) => [
        String(stat.id),
        {
            id: stat.id,
            meta_id: "meta-2026",
            name: stat.name,
            slug: stat.slug,
            core_cards: null,
            core_threshold: stat.id === 1 ? 0.7 : 0.65,
            core_computed_at: "2026-05-15T00:00:00Z",
        },
    ]),
);

const tournaments = Array.from({ length: 25 }, (_, index) => ({
    id: `tour-${String(index + 1).padStart(2, "0")}`,
    name: index === 0 ? "Worlds Warmup Regional" : `Tournament ${String(index + 1).padStart(2, "0")}`,
    game: "pokemon",
    format_code: "STD",
    meta_id: "meta-2026",
    meta_name: "Worlds 2026",
    date: `2026-05-${String((index % 28) + 1).padStart(2, "0")}`,
    players: 32 + index * 4,
    is_online: index % 2 === 1,
    has_decklists: true,
    organizer_name: index === 0 ? "Celadon League" : `Organizer ${index + 1}`,
    winner_archetype: archetypeNames[index % 6],
}));

const standingsByTournamentId = {
    "tour-01": [
        {
            standing: 1,
            wins: 12,
            losses: 1,
            ties: 1,
            player_id: "p-ash",
            player_name: "Ash Ketchum",
            decklist_id: 101,
            archetype_id: 1,
            archetype_name: "Charizard ex",
            archetype_slug: "charizard-ex",
        },
        {
            standing: 2,
            wins: 11,
            losses: 2,
            ties: 1,
            player_id: "p-misty",
            player_name: "Misty Waterflower",
            decklist_id: 102,
            archetype_id: 2,
            archetype_name: "Gardevoir ex",
            archetype_slug: "gardevoir-ex",
        },
        {
            standing: 3,
            wins: 10,
            losses: 3,
            ties: 1,
            player_id: "p-brock",
            player_name: "Brock Harrison",
            decklist_id: null,
            archetype_id: null,
            archetype_name: null,
            archetype_slug: null,
        },
    ],
    "tour-02": [],
};

const playersByNickname = new Map([
    [
        "ash ketchum",
        {
            id: "p-ash",
            name: "Ash Ketchum",
            history: [
                {
                    tournament_id: "tour-01",
                    event_name: "Worlds Warmup Regional",
                    date: "2026-05-01",
                    players: 32,
                    placement: 1,
                    decklist_id: 101,
                    archetype_id: 1,
                    archetype_name: "Charizard ex",
                    archetype_slug: "charizard-ex",
                },
                {
                    tournament_id: "tour-02",
                    event_name: "Tournament 02",
                    date: "2026-05-02",
                    players: 36,
                    placement: 0,
                    decklist_id: null,
                    archetype_id: null,
                    archetype_name: null,
                    archetype_slug: null,
                },
            ],
        },
    ],
]);

const decklistsById = new Map([
    [
        101,
        {
            id: 101,
            tournament_id: "tour-01",
            tournament_name: "Worlds Warmup Regional",
            date: "2026-05-01",
            player_id: "p-ash",
            player_name: "Ash Ketchum",
            archetype_id: 1,
            archetype_name: "Charizard ex",
            archetype_slug: "charizard-ex",
            cards: [
                { name: "Charmander", set: "PAF", number: "7", count: 4, category: "pokemon" },
                { name: "Charizard ex", set: "OBF", number: "125", count: 3, category: "pokemon" },
                { name: "Rare Candy", set: "SVI", number: "191", count: 4, category: "trainer" },
                { name: "Fire Energy", set: "SVE", number: "2", count: 6, category: "energy" },
            ],
        },
    ],
]);

const cardStatsByArchetypeId = {
    "1": [
        {
            name: "Charmander",
            set: "PAF",
            number: "7",
            category: "pokemon",
            is_core: true,
            deck_count: 18,
            total_decklists: 20,
            presence: 0.9,
            modal_count: 4,
            count_distribution: { "4": 0.9 },
        },
        {
            name: "Charizard ex",
            set: "OBF",
            number: "125",
            category: "pokemon",
            is_core: true,
            deck_count: 20,
            total_decklists: 20,
            presence: 1,
            modal_count: 3,
            count_distribution: { "3": 1 },
        },
        {
            name: "Rare Candy",
            set: "SVI",
            number: "191",
            category: "trainer",
            is_core: true,
            deck_count: 18,
            total_decklists: 20,
            presence: 0.9,
            modal_count: 4,
            count_distribution: { "4": 0.9 },
        },
        {
            name: "Fire Energy",
            set: "SVE",
            number: "2",
            category: "energy",
            is_core: true,
            deck_count: 20,
            total_decklists: 20,
            presence: 1,
            modal_count: 6,
            count_distribution: { "6": 0.55, "7": 0.45 },
        },
        {
            name: "Buddy-Buddy Poffin",
            set: "TEF",
            number: "144",
            category: "trainer",
            is_core: false,
            deck_count: 12,
            total_decklists: 20,
            presence: 0.6,
            modal_count: 2,
            count_distribution: { "2": 0.4, "3": 0.2 },
        },
        {
            name: "Defiance Band",
            set: "PAL",
            number: "169",
            category: "trainer",
            is_core: false,
            deck_count: 4,
            total_decklists: 20,
            presence: 0.2,
            modal_count: 1,
            count_distribution: { "1": 0.2 },
        },
    ],
    "25": [
        {
            name: "Dragonite ex",
            set: "DRG",
            number: "5",
            category: "pokemon",
            is_core: true,
            deck_count: 8,
            total_decklists: 10,
            presence: 0.8,
            modal_count: 3,
            count_distribution: { "3": 0.8 },
        },
    ],
};

const matchups = [
    makeMatchup(1, 2, 24, 14, 8, 2),
    makeMatchup(1, 3, 20, 11, 8, 1),
    makeMatchup(1, 4, 18, 9, 8, 1),
    makeMatchup(1, 5, 16, 7, 8, 1),
    makeMatchup(1, 6, 14, 10, 3, 1),
    makeMatchup(1, 7, 22, 15, 6, 1),
    makeMatchup(2, 3, 19, 10, 7, 2),
    makeMatchup(2, 4, 17, 8, 8, 1),
];

const server = http.createServer((req, res) => {
    const requestUrl = new URL(req.url ?? "/", `http://${req.headers.host}`);
    const { pathname, searchParams } = requestUrl;

    if (pathname === "/health") {
        return sendText(res, 200, "ok");
    }

    if (!pathname.startsWith("/api/")) {
        return sendJson(res, 404, { message: "Not found" });
    }

    if (pathname === "/api/metas") {
        return sendJson(res, 200, metas);
    }

    if (pathname === "/api/tournaments") {
        const metaId = searchParams.get("meta_id") ?? "meta-2026";
        if (metaId === "meta-error") {
            return sendJson(res, 500, { message: "Tournament index offline" });
        }
        if (metaId === "meta-empty") {
            return sendJson(res, 200, paginate([], searchParams));
        }

        let items = tournaments.filter((tournament) => tournament.meta_id === metaId);
        const source = searchParams.get("source");
        if (source === "online") {
            items = items.filter((tournament) => tournament.is_online);
        } else if (source === "offline") {
            items = items.filter((tournament) => !tournament.is_online);
        }

        const minPlayers = Number(searchParams.get("min_players") ?? "0");
        items = items.filter((tournament) => tournament.players >= minPlayers);

        const dateFrom = searchParams.get("date_from");
        if (dateFrom) {
            items = items.filter((tournament) => tournament.date >= dateFrom);
        }

        const dateTo = searchParams.get("date_to");
        if (dateTo) {
            items = items.filter((tournament) => tournament.date <= dateTo);
        }

        const winnerArchetype = searchParams.get("winner_archetype");
        if (winnerArchetype) {
            items = items.filter(
                (tournament) => slugify(tournament.winner_archetype ?? "") === winnerArchetype,
            );
        }

        return sendJson(res, 200, paginate(items, searchParams));
    }

    if (pathname.startsWith("/api/tournaments/")) {
        const id = pathname.split("/").pop();
        if (id === "missing") {
            return sendJson(res, 404, { message: "Tournament not found" });
        }
        if (id === "error") {
            return sendText(res, 500, "Tournament sync failed");
        }

        const tournament = tournaments.find((entry) => entry.id === id);
        if (!tournament) {
            return sendJson(res, 404, { message: "Tournament not found" });
        }

        return sendJson(res, 200, {
            ...tournament,
            standings: standingsByTournamentId[id] ?? standingsByTournamentId["tour-01"],
        });
    }

    if (pathname === "/api/archetypes/stats") {
        const metaId = searchParams.get("meta_id") ?? "meta-2026";
        if (metaId === "meta-error") {
            return sendJson(res, 500, { message: "Archetype stats unavailable" });
        }
        if (metaId === "meta-empty") {
            return sendJson(res, 200, []);
        }
        return sendJson(res, 200, archetypeStats);
    }

    if (pathname.startsWith("/api/archetypes/") && pathname.endsWith("/card-stats")) {
        const id = pathname.split("/")[3];
        if (!archetypeById.has(id)) {
            return sendJson(res, 404, { message: "Archetype not found" });
        }
        return sendJson(res, 200, cardStatsByArchetypeId[id] ?? []);
    }

    if (pathname.startsWith("/api/archetypes/") && pathname.endsWith("/variants")) {
        const id = pathname.split("/")[3];
        if (!archetypeById.has(id)) {
            return sendJson(res, 404, { message: "Archetype not found" });
        }
        return sendJson(res, 200, []);
    }

    if (pathname === "/api/matchups/stats") {
        const metaId = searchParams.get("meta_id") ?? "meta-2026";
        if (metaId === "meta-error") {
            return sendJson(res, 500, { message: "Matchup stats unavailable" });
        }
        if (metaId === "meta-empty") {
            return sendJson(res, 200, paginate([], searchParams));
        }

        let items = [...matchups];
        const archetypeId = searchParams.get("archetype_id");
        if (archetypeId) {
            items = items.filter(
                (stat) =>
                    String(stat.archetype.id) === archetypeId ||
                    String(stat.opponent.id) === archetypeId,
            );
        }

        const includeMirrors = searchParams.get("include_mirrors");
        if (includeMirrors === "false") {
            items = items.filter((stat) => stat.archetype.id !== stat.opponent.id);
        }

        const minMatches = Number(searchParams.get("min_matches") ?? "0");
        items = items.filter((stat) => stat.matches >= minMatches);

        return sendJson(res, 200, paginate(items, searchParams));
    }

    if (pathname.startsWith("/api/players/")) {
        const nickname = decodeURIComponent(pathname.split("/").pop());
        const player = playersByNickname.get(nickname.toLowerCase());
        if (!player) {
            return sendJson(res, 404, { message: "Player not found" });
        }
        return sendJson(res, 200, player);
    }

    if (pathname.startsWith("/api/decklists/")) {
        const id = Number(pathname.split("/").pop());
        const decklist = decklistsById.get(id);
        if (!decklist) {
            return sendJson(res, 404, { message: "Decklist not found" });
        }
        return sendJson(res, 200, decklist);
    }

    if (pathname.startsWith("/api/archetypes/")) {
        const id = pathname.split("/").pop();
        if (id === "missing" || !archetypeById.has(id)) {
            return sendJson(res, 404, { message: "Archetype not found" });
        }
        return sendJson(res, 200, archetypeById.get(id));
    }

    return sendJson(res, 404, { message: "Not found" });
});

server.listen(port, host, () => {
    console.log(`Mock API server listening on http://${host}:${port}`);
});

function slugify(value) {
    return value
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "");
}

function makeMatchup(archetypeId, opponentId, matches, wins, losses, ties) {
    const archetype = archetypeStats[archetypeId - 1];
    const opponent = archetypeStats[opponentId - 1];
    return {
        archetype: {
            id: archetype.id,
            name: archetype.name,
            slug: archetype.slug,
        },
        opponent: {
            id: opponent.id,
            name: opponent.name,
            slug: opponent.slug,
        },
        matches,
        wins,
        losses,
        ties,
        score_rate: (wins + ties * 0.5) / matches,
        win_rate: wins + losses > 0 ? wins / (wins + losses) : null,
    };
}

function paginate(items, searchParams) {
    const page = Math.max(1, Number(searchParams.get("page") ?? "1") || 1);
    const pageSize = Math.max(1, Number(searchParams.get("page_size") ?? "20") || 20);
    const total = items.length;
    const totalPages = Math.max(1, Math.ceil(total / pageSize));
    const safePage = Math.min(page, totalPages);
    const start = (safePage - 1) * pageSize;
    const pageItems = items.slice(start, start + pageSize);

    return {
        total,
        page: safePage,
        page_size: pageSize,
        total_pages: totalPages,
        prev_page: safePage > 1 ? safePage - 1 : 0,
        next_page: safePage < totalPages ? safePage + 1 : 0,
        prev_url: safePage > 1 ? `?page=${safePage - 1}` : null,
        next_url: safePage < totalPages ? `?page=${safePage + 1}` : null,
        items: pageItems,
    };
}

function sendJson(res, status, data) {
    res.writeHead(status, {
        "content-type": "application/json; charset=utf-8",
        "access-control-allow-origin": "*",
    });
    res.end(JSON.stringify(data));
}

function sendText(res, status, data) {
    res.writeHead(status, {
        "content-type": "text/plain; charset=utf-8",
        "access-control-allow-origin": "*",
    });
    res.end(data);
}
