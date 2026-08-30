import type { ArchetypeStat, Meta, Tournament } from "@/lib/types";
import { getArchetypeStats, getMetas, getTournaments } from "@/lib/api";
import Hero from "@/components/hero";
import Card from "@/components/card";
import Pagination from "@/components/pagination";
import FilterForm from "@/components/filter-form";
import TournamentsTable from "./TournamentsTable";

type SearchParams = {
    meta_id?: string;
    source?: string;
    min_players?: string;
    date_from?: string;
    date_to?: string;
    winner_archetype?: string;
    event_name?: string;
    sort_by?: string;
    sort_dir?: string;
    page?: string;
};

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

function TournamentFilters({
    metas,
    activeMeta,
    archetypes,
    source,
    minPlayers,
    dateFrom,
    dateTo,
    winnerArchetype,
    eventName,
}: {
    metas: Meta[];
    activeMeta: Meta | null;
    archetypes: ArchetypeStat[];
    source: string;
    minPlayers: number;
    dateFrom: string;
    dateTo: string;
    winnerArchetype: string;
    eventName: string;
}) {
    return (
        <FilterForm
            style={{ display: "flex", flexDirection: "column", gap: "14px" }}
        >
            <div className="selector selector--stack">
                <div className="selector__field" style={{ flex: 1 }}>
                    <p className="eyebrow">Event name</p>
                    <label className="sr-only" htmlFor="event_name">
                        Search event name
                    </label>
                    <input
                        id="event_name"
                        name="event_name"
                        type="search"
                        placeholder="Search events…"
                        defaultValue={eventName}
                        style={{ width: "100%", minWidth: "auto" }}
                    />
                </div>

                <div className="selector__field">
                    <p className="eyebrow">Meta</p>
                    <label className="sr-only" htmlFor="meta_id">
                        Select meta
                    </label>
                    <select
                        id="meta_id"
                        name="meta_id"
                        defaultValue={activeMeta?.id ?? ""}
                    >
                        {metas.map((meta) => (
                            <option key={meta.id} value={meta.id}>
                                {meta.name}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="selector__field">
                    <p className="eyebrow">Source</p>
                    <label className="sr-only" htmlFor="source">
                        Filter by source
                    </label>
                    <select id="source" name="source" defaultValue={source}>
                        <option value="">All sources</option>
                        <option value="online">Online</option>
                        <option value="offline">In person</option>
                    </select>
                </div>

                <div className="selector__field">
                    <p className="eyebrow">Minimum players</p>
                    <label className="sr-only" htmlFor="min_players">
                        Minimum players
                    </label>
                    <input
                        id="min_players"
                        name="min_players"
                        type="number"
                        min={0}
                        defaultValue={minPlayers}
                    />
                </div>
            </div>

            <div className="selector selector--stack">
                <div className="selector__field">
                    <p className="eyebrow">From date</p>
                    <label className="sr-only" htmlFor="date_from">
                        From date
                    </label>
                    <input
                        id="date_from"
                        name="date_from"
                        type="date"
                        defaultValue={dateFrom}
                    />
                </div>

                <div className="selector__field">
                    <p className="eyebrow">To date</p>
                    <label className="sr-only" htmlFor="date_to">
                        To date
                    </label>
                    <input
                        id="date_to"
                        name="date_to"
                        type="date"
                        defaultValue={dateTo}
                    />
                </div>

                <div className="selector__field">
                    <p className="eyebrow">Winner archetype</p>
                    <label className="sr-only" htmlFor="winner_archetype">
                        Filter by winner archetype
                    </label>
                    <select
                        id="winner_archetype"
                        name="winner_archetype"
                        defaultValue={winnerArchetype}
                    >
                        <option value="">All archetypes</option>
                        {archetypes.map((archetype) => (
                            <option key={archetype.id} value={archetype.slug}>
                                {archetype.name}
                            </option>
                        ))}
                    </select>
                </div>

                <button type="submit">Apply filters</button>
            </div>
        </FilterForm>
    );
}

export default async function TournamentsPage({
    searchParams,
}: {
    searchParams: Promise<SearchParams>;
}) {
    const params = await searchParams;
    const metas = await getMetas().catch(() => [] as Meta[]);
    const activeMeta =
        metas.find((meta) => meta.id === params.meta_id) ?? metas[0] ?? null;

    const source =
        params.source === "online" || params.source === "offline"
            ? params.source
            : "";
    const parsedMinPlayers = Number.parseInt(params.min_players ?? "0", 10);
    const minPlayers =
        Number.isFinite(parsedMinPlayers) && parsedMinPlayers >= 0
            ? parsedMinPlayers
            : 0;
    const dateFrom = params.date_from ?? "";
    const dateTo = params.date_to ?? "";
    const winnerArchetype = params.winner_archetype ?? "";
    const eventName = params.event_name ?? "";
    const sortBy = params.sort_by ?? "";
    const sortDir = params.sort_dir === "asc" ? "asc" : "desc";
    const page = Number.parseInt(params.page ?? "1", 10) || 1;
    const pageSize = 20;

    const archetypes = activeMeta
        ? await getArchetypeStats(activeMeta.id).catch(
              () => [] as ArchetypeStat[],
          )
        : [];

    const tournamentPage = activeMeta
        ? await getTournaments({
              metaId: activeMeta.id,
              minPlayers,
              source: source || undefined,
              dateFrom: dateFrom || undefined,
              dateTo: dateTo || undefined,
              winnerArchetype: winnerArchetype || undefined,
              eventName: eventName || undefined,
              sortBy: sortBy || undefined,
              sortDir,
              page,
              pageSize,
          }).catch(() => ({
              total: 0,
              page,
              page_size: pageSize,
              total_pages: 1,
              prev_page: 0,
              next_page: 0,
              items: [] as Tournament[],
          }))
        : {
              total: 0,
              page,
              page_size: pageSize,
              total_pages: 1,
              prev_page: 0,
              next_page: 0,
              items: [] as Tournament[],
          };
    const tournaments = tournamentPage.items;
    const totalPages = Math.max(
        1,
        Math.ceil(tournamentPage.total / tournamentPage.page_size),
    );

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="META Radar - Events"
                    title="Tournament Explorer"
                    lede="Browse every synced event for a meta, then open one
                        to see its full leaderboard, standings, score, and
                        archetype per player."
                    meta={
                        <>
                            {activeMeta ? (
                                <span className="pill">{activeMeta.name}</span>
                            ) : null}
                        </>
                    }
                />

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Filters</p>
                            <h2>Search Tournaments</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {tournamentPage.total.toLocaleString()} events
                        </span>
                    }
                >
                    {metas.length > 0 ? (
                        <TournamentFilters
                            metas={metas}
                            activeMeta={activeMeta}
                            archetypes={archetypes}
                            source={source}
                            minPlayers={minPlayers}
                            dateFrom={dateFrom}
                            dateTo={dateTo}
                            winnerArchetype={winnerArchetype}
                            eventName={eventName}
                        />
                    ) : (
                        <EmptyState
                            title="No Metas yet"
                            copy="Seed a meta before tournaments can be loaded."
                        />
                    )}
                </Card>

                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Events</p>
                            <h2>
                                {activeMeta
                                    ? activeMeta.name
                                    : "No meta loaded"}
                            </h2>
                        </>
                    }
                >
                    {tournaments.length > 0 ? (
                        <>
                            <TournamentsTable tournaments={tournaments} />
                            <div className="pagination">
                                <div
                                    style={{
                                        display: "flex",
                                        gap: "8px",
                                        marginTop: "12px",
                                        alignItems: "center",
                                    }}
                                >
                                    <span className="muted">
                                        Page {tournamentPage.page} of{" "}
                                        {totalPages}
                                    </span>
                                </div>

                                <div
                                    style={{
                                        display: "flex",
                                        gap: "8px",
                                        marginTop: "8px",
                                    }}
                                >
                                    <Pagination
                                        page={tournamentPage.page}
                                        totalPages={totalPages}
                                    />
                                </div>
                            </div>
                        </>
                    ) : (
                        <EmptyState
                            title="No tournaments found"
                            copy="Once ingestion has synced events for this meta, they'll appear here."
                        />
                    )}
                </Card>
            </div>
        </main>
    );
}
