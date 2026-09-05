import type { ArchetypeStat, Meta, Tournament } from "@/lib/types";
import Link from "next/link";
import Hero from "@/components/hero";
import Card from "@/components/card";

import { getArchetypeStats, getMetas, getTournaments } from "@/lib/api";
import { LiveTournamentsTable, TopArchetypesTable } from "./HomeTables";

type SearchParams = {
    meta_id?: string;
};

// Static placeholders until each set's release date is confirmed and
// wired up to a real data source; update these by hand each set cycle.
// Ordered soonest-first.
const NEXT_SET_RELEASES = [
    { name: "30th Anniversary", date: "2026-09-16" },
    { name: "Delta Reign", date: "2026-11-06" },
];

// Minimum sample size before an archetype's win rate is considered stable
// enough to surface as the homepage's "top win-rate deck".
const MIN_MATCHES_FOR_TOP_WIN_RATE = 50;

function formatCardDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
    }).format(new Date(value));
}

function formatCardPercent(value: number | null) {
    if (value === null) {
        return null;
    }

    return `${Math.round(value * 1000) / 10}%`;
}

function TopStatCard({
    label,
    value,
    detail,
}: {
    label: string;
    value: string;
    detail: string;
}) {
    return (
        <Card className="card--tight">
            <p className="eyebrow">{label}</p>
            <p className="stat-value">{value}</p>
            <p className="muted">{detail}</p>
        </Card>
    );
}

function UpcomingSetsCard({
    label,
    releases,
}: {
    label: string;
    releases: { name: string; date: string }[];
}) {
    const [primary, ...rest] = releases;

    return (
        <Card className="card--tight">
            <p className="eyebrow">{label}</p>
            <p className="stat-value">{primary.name}</p>
            <p className="muted">{formatCardDate(primary.date)}</p>
            {rest.map((release) => (
                <div key={release.name} className="stat-release-next">
                    <p className="stat-value stat-value--sm">{release.name}</p>
                    <p className="muted">{formatCardDate(release.date)}</p>
                </div>
            ))}
        </Card>
    );
}

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

function MetaSelector({
    metas,
    activeMeta,
}: {
    metas: Meta[];
    activeMeta: Meta | null;
}) {
    return (
        <form className="selector" method="get">
            <div>
                <p className="eyebrow">Current meta</p>
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
            <button type="submit">Load meta</button>
        </form>
    );
}

export default async function Home({
    searchParams,
}: {
    searchParams: Promise<SearchParams>;
}) {
    const params = await searchParams;
    const metas = await getMetas().catch(() => [] as Meta[]);
    const activeMeta =
        metas.find((meta) => meta.id === params.meta_id) ?? metas[0] ?? null;

    const [tournamentPage, archetypes, doomTournamentPage] = await Promise.all([
        activeMeta
            ? getTournaments({
                  metaId: activeMeta.id,
                  minPlayers: 32,
                  page: 1,
                  pageSize: 8,
              }).catch(() => ({ items: [] as Tournament[] }))
            : Promise.resolve({ items: [] as Tournament[] }),
        activeMeta
            ? getArchetypeStats(activeMeta.id).catch(
                  () => [] as ArchetypeStat[],
              )
            : Promise.resolve([] as ArchetypeStat[]),
        // "Doom" tournaments are identified by having "Doom" somewhere in
        // the tournament name, not by meta, so this is a separate,
        // meta-independent lookup for the most recent one (no min player
        // floor, since Doom's events don't necessarily hit the 32+
        // threshold used for the events table below). The card surfaces the
        // winning archetype, not the tournament name itself.
        getTournaments({
            eventName: "DOOM",
            minPlayers: 0,
            sortBy: "date",
            sortDir: "desc",
            page: 1,
            pageSize: 1,
        }).catch(() => ({ items: [] as Tournament[] })),
    ]);

    // Archetype stats come back sorted by deck_count DESC (see backend
    // ArchetypeStats handler), so the first entry is already the top played
    // deck; the top win rate deck needs a separate pass since win rate and
    // play count don't necessarily move together. Archetypes with a small
    // sample size are excluded so a lucky small deck doesn't outrank a
    // proven high-volume one.
    const topPlayedArchetype = archetypes[0] ?? null;
    const topWinRateArchetype = archetypes.reduce<ArchetypeStat | null>(
        (best, current) => {
            if (
                current.win_rate === null ||
                current.matches < MIN_MATCHES_FOR_TOP_WIN_RATE
            ) {
                return best;
            }
            if (best === null || current.win_rate > (best.win_rate ?? -1)) {
                return current;
            }
            return best;
        },
        null,
    );
    const liveTournaments = tournamentPage.items;
    const latestTournament = doomTournamentPage.items[0] ?? null;

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="META Radar - Pokémon TCG"
                    title="Meta Radar Dashboard"
                    lede="Analyze tournaments and deck performance,
                        then jump into directional matchup results."
                    meta={
                        <>
                            <span className="pill">PBL+ Tracked</span>
                            <span className="pill pill--soft">v1.0.1</span>
                        </>
                    }
                />

                <section className="grid grid--summary">
                    <TopStatCard
                        label="Top win rate deck"
                        value={topWinRateArchetype?.name ?? "—"}
                        detail={
                            topWinRateArchetype
                                ? `${formatCardPercent(topWinRateArchetype.win_rate) ?? "—"} win-rate across ${topWinRateArchetype.matches.toLocaleString()} matches`
                                : `No archetype with ${MIN_MATCHES_FOR_TOP_WIN_RATE}+ matches yet.`
                        }
                    />
                    <TopStatCard
                        label="Top played deck"
                        value={topPlayedArchetype?.name ?? "—"}
                        detail={
                            topPlayedArchetype
                                ? `${topPlayedArchetype.deck_count.toLocaleString()} decklists in meta`
                                : "No archetype data yet."
                        }
                    />
                    <TopStatCard
                        label="Latest Doom winner"
                        value={latestTournament?.winner_archetype ?? "—"}
                        detail={
                            latestTournament
                                ? `${latestTournament.players} players · ${formatCardDate(latestTournament.date)}`
                                : "No Doom tournaments synced yet."
                        }
                    />
                    <UpcomingSetsCard
                        label="Upcoming set releases"
                        releases={NEXT_SET_RELEASES}
                    />
                </section>

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Meta Selection</p>
                            <h2>
                                {activeMeta
                                    ? activeMeta.name
                                    : "No meta loaded"}
                            </h2>
                        </>
                    }
                >
                    {metas.length > 0 ? (
                        <MetaSelector metas={metas} activeMeta={activeMeta} />
                    ) : (
                        <EmptyState
                            title="No Metas yet"
                            copy="Seed a specific meta before the dashboard can populate tournaments and archetypes."
                        />
                    )}
                </Card>

                <section className="grid grid--two">
                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Recent Tournaments</p>
                                <h2>Latest Events</h2>
                            </>
                        }
                        headingMeta={<span className="muted">32+ players</span>}
                    >
                        {liveTournaments.length > 0 ? (
                            <LiveTournamentsTable
                                tournaments={liveTournaments}
                            />
                        ) : (
                            <EmptyState
                                title="No tournaments found"
                                copy="Once ingestion has synced events for this meta, the latest tournaments will appear here."
                            />
                        )}
                    </Card>

                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Archetype Stats</p>
                                <h2>Trending Decks</h2>
                            </>
                        }
                        headingMeta={
                            <Link
                                className="pill pill--soft"
                                href={
                                    activeMeta
                                        ? `/matchups?meta_id=${activeMeta.id}`
                                        : "/matchups"
                                }
                            >
                                View Matchups
                            </Link>
                        }
                    >
                        {archetypes.length > 0 ? (
                            <TopArchetypesTable
                                archetypes={archetypes.slice(0, 10)}
                                activeMetaId={activeMeta?.id ?? null}
                            />
                        ) : (
                            <EmptyState
                                title="No archetype stats yet"
                                copy="Run the clustering and pairings pipeline to populate archetype performance data."
                            />
                        )}
                    </Card>
                </section>
            </div>
        </main>
    );
}
