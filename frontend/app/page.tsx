import type { ArchetypeStat, Meta, Tournament } from "@/lib/types";

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
        <section className="card card--tight">
            <p className="eyebrow">{label}</p>
            <p className="stat-value">{value}</p>
            <p className="muted">{detail}</p>
        </section>
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
    searchParams?: SearchParams;
}) {
    const metas = await getMetas().catch(() => [] as Meta[]);
    const activeMeta =
        metas.find((meta) => meta.id === searchParams?.meta_id) ??
        metas[0] ??
        null;

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
                <header className="hero card">
                    <div>
                        <p className="eyebrow">META Radar - Pokémon TCG</p>
                        <h1>
                            Dashboard for trending decks and archetype
                            performance.
                        </h1>
                        <p className="lede">
                            Start with the current meta, then drill into
                            tournaments and deck performance. Matchups and deep
                            archetype pages can land next.
                        </p>
                    </div>

                    <div className="hero__meta">
                        <span className="pill">Next.js App Router</span>
                        <span className="pill pill--soft">Core views only</span>
                    </div>
                </header>

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

                <section className="card section">
                    <div className="section__heading">
                        <div>
                            <p className="eyebrow">Meta selection</p>
                            <h2>
                                {activeMeta
                                    ? activeMeta.name
                                    : "No meta loaded"}
                            </h2>
                        </div>
                        {activeMeta ? (
                            <span className="pill">
                                {activeMeta.format_code}
                            </span>
                        ) : null}
                    </div>

                    {metas.length > 0 ? (
                        <MetaSelector metas={metas} activeMeta={activeMeta} />
                    ) : (
                        <EmptyState
                            title="No metas yet"
                            copy="Seed a specific meta before the dashboard can populate tournaments and archetypes."
                        />
                    )}
                </section>

                <section className="grid grid--two">
                    <article className="card section">
                        <div className="section__heading">
                            <div>
                                <p className="eyebrow">Recent tournaments</p>
                                <h2>Latest events</h2>
                            </div>
                            <span className="muted">64+ players only</span>
                        </div>

                        {liveTournaments.length > 0 ? (
                            <div className="table-wrap">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Event</th>
                                            <th>Date</th>
                                            <th>Players</th>
                                            <th>Source</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {liveTournaments.map((tournament) => (
                                            <tr key={tournament.id}>
                                                <td>
                                                    <div className="table-title">
                                                        {tournament.name}
                                                    </div>
                                                    <div className="muted tiny">
                                                        {tournament.format_code}
                                                    </div>
                                                </td>
                                                <td>
                                                    {formatDate(
                                                        tournament.date,
                                                    )}
                                                </td>
                                                <td>
                                                    {tournament.players.toLocaleString()}
                                                </td>
                                                <td>
                                                    <span
                                                        className={`badge ${tournament.is_online ? "badge--online" : ""}`}
                                                    >
                                                        {tournament.is_online
                                                            ? "Online"
                                                            : "In person"}
                                                    </span>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        ) : (
                            <EmptyState
                                title="No tournaments found"
                                copy="Once ingestion has synced events for this meta, the latest tournaments will appear here."
                            />
                        )}
                    </article>

                    <article className="card section">
                        <div className="section__heading">
                            <div>
                                <p className="eyebrow">Archetype stats</p>
                                <h2>Performance snapshot</h2>
                            </div>
                            <span className="muted">
                                Deck count, standing, and win rate
                            </span>
                        </div>

                        {archetypes.length > 0 ? (
                            <div className="table-wrap">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Archetype</th>
                                            <th>Decks</th>
                                            <th>Avg standing</th>
                                            <th>Win rate</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {archetypes.slice(0, 10).map((stat) => (
                                            <tr key={stat.id}>
                                                <td>
                                                    <div className="table-title">
                                                        {stat.name}
                                                    </div>
                                                    <div className="muted tiny">
                                                        {stat.slug}
                                                    </div>
                                                </td>
                                                <td>
                                                    {stat.deck_count.toLocaleString()}
                                                </td>
                                                <td>
                                                    {formatStanding(
                                                        stat.avg_standing,
                                                    )}
                                                </td>
                                                <td>
                                                    {formatPercent(
                                                        stat.win_rate,
                                                    )}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        ) : (
                            <EmptyState
                                title="No archetype stats yet"
                                copy="Run the clustering and pairings pipeline to populate archetype performance data."
                            />
                        )}
                    </article>
                </section>
            </div>
        </main>
    );
}
