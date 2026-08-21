import type { ArchetypeStat, Meta, Tournament } from "@/lib/types";
import Link from "next/link";
import Hero from "@/components/hero";
import Table from "@/components/table";
import Card from "@/components/card";

import { getArchetypeStats, getMetas, getTournaments } from "@/lib/api";

type SearchParams = {
    meta_id?: string;
};

function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    }).format(new Date(value));
}

function formatPercent(value: number | null) {
    if (value === null) {
        return "—";
    }

    return `${Math.round(value * 1000) / 10}%`;
}

function formatStanding(value: number | null) {
    if (value === null) {
        return "—";
    }

    return Math.round(value).toLocaleString();
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

    const [tournaments, archetypes] = activeMeta
        ? await Promise.all([
              getTournaments({ metaId: activeMeta.id, minPlayers: 64 }).catch(
                  () => [] as Tournament[],
              ),
              getArchetypeStats(activeMeta.id).catch(
                  () => [] as ArchetypeStat[],
              ),
          ])
        : [[], []];

    const topArchetype = archetypes[0] ?? null;
    const liveTournaments = tournaments.slice(0, 8);

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
                            <span className="pill">Next.js App Router</span>
                            <span className="pill pill--soft">
                                Core views only
                            </span>
                        </>
                    }
                />

                <section className="grid grid--summary">
                    <TopStatCard
                        label="Metas tracked"
                        value={metas.length.toLocaleString()}
                        detail="Pulled from the backend metas table."
                    />
                    <TopStatCard
                        label="Recent tournaments"
                        value={liveTournaments.length.toLocaleString()}
                        detail="Filtered to 64+ player events for the selected meta."
                    />
                    <TopStatCard
                        label="Top archetype"
                        value={topArchetype?.name ?? "—"}
                        detail={
                            topArchetype
                                ? `${topArchetype.deck_count} decklists in meta`
                                : "No archetype data yet."
                        }
                    />
                    <TopStatCard
                        label="Matchup layer"
                        value="Next"
                        detail="Reserved for the archetype-vs-archetype view."
                    />
                </section>

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Meta selection</p>
                            <h2>
                                {activeMeta
                                    ? activeMeta.name
                                    : "No meta loaded"}
                            </h2>
                        </>
                    }
                    headingMeta={
                        activeMeta ? (
                            <span className="pill">
                                {activeMeta.format_code}
                            </span>
                        ) : null
                    }
                >
                    {metas.length > 0 ? (
                        <MetaSelector metas={metas} activeMeta={activeMeta} />
                    ) : (
                        <EmptyState
                            title="No metas yet"
                            copy="Seed a specific meta before the dashboard can populate tournaments and archetypes."
                        />
                    )}
                </Card>

                <section className="grid grid--two">
                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Recent tournaments</p>
                                <h2>Latest events</h2>
                            </>
                        }
                        headingMeta={
                            <span className="muted">64+ players only</span>
                        }
                    >
                        {liveTournaments.length > 0 ? (
                            <Table
                                columns={[
                                    {
                                        key: "event",
                                        label: "Event",
                                        render: (t: Tournament) => (
                                            <Link
                                                className="table-link"
                                                href={`/tournaments/${t.id}`}
                                            >
                                                <div className="table-title">
                                                    {t.name}
                                                </div>
                                                <div className="muted tiny">
                                                    {t.meta_name}
                                                </div>
                                            </Link>
                                        ),
                                    },
                                    {
                                        key: "date",
                                        label: "Date",
                                        render: (t: Tournament) =>
                                            formatDate(t.date),
                                    },
                                    {
                                        key: "players",
                                        label: "Players",
                                        render: (t: Tournament) =>
                                            t.players.toLocaleString(),
                                    },
                                    {
                                        key: "source",
                                        label: "Source",
                                        render: (t: Tournament) => (
                                            <span
                                                className={`badge ${t.is_online ? "badge--online" : ""}`}
                                            >
                                                {t.is_online
                                                    ? "Online"
                                                    : "In person"}
                                            </span>
                                        ),
                                    },
                                ]}
                                rows={liveTournaments}
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
                                <p className="eyebrow">Archetype stats</p>
                                <h2>Performance snapshot</h2>
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
                                View matchups
                            </Link>
                        }
                    >
                        {archetypes.length > 0 ? (
                            <Table
                                columns={[
                                    {
                                        key: "archetype",
                                        label: "Archetype",
                                        render: (stat: ArchetypeStat) =>
                                            activeMeta ? (
                                                <Link
                                                    className="table-link"
                                                    href={`/matchups?meta_id=${activeMeta.id}&archetype_id=${stat.id}`}
                                                >
                                                    <div className="table-title">
                                                        {stat.name}
                                                    </div>
                                                </Link>
                                            ) : (
                                                <div className="table-title">
                                                    {stat.name}
                                                </div>
                                            ),
                                    },
                                    {
                                        key: "decks",
                                        label: "Decks",
                                        render: (s: ArchetypeStat) =>
                                            s.deck_count.toLocaleString(),
                                    },
                                    {
                                        key: "avg",
                                        label: "Avg standing",
                                        render: (s: ArchetypeStat) =>
                                            formatStanding(s.avg_standing),
                                    },
                                    {
                                        key: "win",
                                        label: "Win rate",
                                        render: (s: ArchetypeStat) =>
                                            formatPercent(s.win_rate),
                                    },
                                ]}
                                rows={archetypes.slice(0, 10)}
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
