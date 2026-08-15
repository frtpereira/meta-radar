import Link from "next/link";

import type { Meta, Tournament } from "@/lib/types";
import { getMetas, getTournaments } from "@/lib/api";

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
    return (
        <div className="table-wrap">
            <table>
                <thead>
                    <tr>
                        <th>Event</th>
                        <th>Date</th>
                        <th>Players</th>
                        <th>Source</th>
                        <th>Winner archetype</th>
                    </tr>
                </thead>
                <tbody>
                    {tournaments.map((tournament) => (
                        <tr key={tournament.id}>
                            <td>
                                <Link
                                    className="table-link"
                                    href={`/tournaments/${tournament.id}`}
                                >
                                    <div className="table-title">
                                        {tournament.name}
                                    </div>
                                    <div className="muted tiny">
                                        {tournament.format_code}
                                    </div>
                                </Link>
                            </td>
                            <td>{formatDate(tournament.date)}</td>
                            <td>{tournament.players.toLocaleString()}</td>
                            <td>
                                <span
                                    className={`badge ${tournament.is_online ? "badge--online" : ""}`}
                                >
                                    {tournament.is_online
                                        ? "Online"
                                        : "In person"}
                                </span>
                            </td>
                            <td>
                                {tournament.winner_archetype ?? (
                                    <span className="muted tiny">
                                        Unknown
                                    </span>
                                )}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
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
                <header className="hero card">
                    <div>
                        <p className="eyebrow">Tournament Explorer</p>
                        <h1>All events for a given meta.</h1>
                        <p className="lede">
                            Browse every synced event for a meta, then open
                            one to see its full leaderboard, standings, score,
                            and archetype per player.
                        </p>
                    </div>

                    <div className="hero__meta">
                        {activeMeta ? (
                            <span className="pill">{activeMeta.name}</span>
                        ) : null}
                    </div>
                </header>

                <section className="card section">
                    <div className="section__heading">
                        <div>
                            <p className="eyebrow">Filters</p>
                            <h2>Meta selection</h2>
                        </div>
                        <span className="muted">
                            {tournaments.length.toLocaleString()} events
                        </span>
                    </div>

                    {metas.length > 0 ? (
                        <MetaSelector metas={metas} activeMeta={activeMeta} />
                    ) : (
                        <EmptyState
                            title="No metas yet"
                            copy="Seed a meta before tournaments can be loaded."
                        />
                    )}
                </section>

                <section className="card section section--spaced">
                    <div className="section__heading">
                        <div>
                            <p className="eyebrow">Events</p>
                            <h2>
                                {activeMeta
                                    ? activeMeta.name
                                    : "No meta loaded"}
                            </h2>
                        </div>
                    </div>

                    {tournaments.length > 0 ? (
                        <TournamentsTable tournaments={tournaments} />
                    ) : (
                        <EmptyState
                            title="No tournaments found"
                            copy="Once ingestion has synced events for this meta, they'll appear here."
                        />
                    )}
                </section>
            </div>
        </main>
    );
}
