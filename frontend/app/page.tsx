import type { ArchetypeStat, Meta, Tournament } from "@/lib/types";
import Link from "next/link";
import Hero from "@/components/hero";
import Card from "@/components/card";

import { getArchetypeStats, getMetas, getTournaments } from "@/lib/api";
import { LiveTournamentsTable, TopArchetypesTable } from "./HomeTables";

type SearchParams = {
    meta_id?: string;
};

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

    const [tournamentPage, archetypes] = activeMeta
        ? await Promise.all([
              getTournaments({
                  metaId: activeMeta.id,
                  minPlayers: 32,
                  page: 1,
                  pageSize: 8,
              }).catch(() => ({ items: [] as Tournament[] })),
              getArchetypeStats(activeMeta.id).catch(
                  () => [] as ArchetypeStat[],
              ),
          ])
        : [{ items: [] as Tournament[] }, [] as ArchetypeStat[]];

    const topArchetype = archetypes[0] ?? null;
    const liveTournaments = tournamentPage.items;

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
                            <span className="pill">PBL+ Metas</span>
                            <span className="pill pill--soft">[WIP]</span>
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
                        detail="Filtered to 32+ player events for the selected meta."
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
