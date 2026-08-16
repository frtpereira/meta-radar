import Link from "next/link";
import Table from "@/components/table";

import type { Meta, Tournament } from "@/lib/types";
import { getMetas, getTournaments } from "@/lib/api";
import Hero from "@/components/hero";
import Card from "@/components/card";

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
            <button type="submit">Load meta</button>
        </form>
    );
}

function TournamentsTable({ tournaments }: { tournaments: Tournament[] }) {
    const columns = [
        {
            key: "event",
            label: "Event",
            render: (t: Tournament) => (
                <Link className="table-link" href={`/tournaments/${t.id}`}>
                    <div className="table-title">{t.name}</div>
                    <div className="muted tiny">{t.format_code}</div>
                </Link>
            ),
        },
        {
            key: "date",
            label: "Date",
            render: (t: Tournament) => formatDate(t.date),
        },
        {
            key: "players",
            label: "Players",
            render: (t: Tournament) => t.players.toLocaleString(),
        },
        {
            key: "source",
            label: "Source",
            render: (t: Tournament) => (
                <span className={`badge ${t.is_online ? "badge--online" : ""}`}>
                    {t.is_online ? "Online" : "In person"}
                </span>
            ),
        },
        {
            key: "winner_archetype",
            label: "Winner archetype",
            render: (t: Tournament) =>
                t.winner_archetype ?? (
                    <span className="muted tiny">Unknown</span>
                ),
        },
    ];

    return <Table columns={columns} rows={tournaments} />;
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

    const tournaments = activeMeta
        ? await getTournaments({ metaId: activeMeta.id, minPlayers: 0 }).catch(
              () => [] as Tournament[],
          )
        : [];

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="Tournament Explorer"
                    title="All events for a given meta."
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
                            <h2>Meta selection</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {tournaments.length.toLocaleString()} events
                        </span>
                    }
                >
                    {metas.length > 0 ? (
                        <MetaSelector metas={metas} activeMeta={activeMeta} />
                    ) : (
                        <EmptyState
                            title="No metas yet"
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
                        <TournamentsTable tournaments={tournaments} />
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
