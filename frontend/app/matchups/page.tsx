import type { ArchetypeStat, MatchupStat, Meta } from "@/lib/types";

import { getArchetypeStats, getMatchupStats, getMetas } from "@/lib/api";

type SearchParams = {
    meta_id?: string;
    archetype_id?: string;
    min_matches?: string;
    include_mirrors?: string;
};

function formatPercent(value: number | null) {
    if (value === null) {
        return "—";
    }

    return `${Math.round(value * 1000) / 10}%`;
}

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

function MatchupFilters({
    metas,
    activeMeta,
    archetypes,
    selectedArchetypeId,
    minMatches,
    includeMirrors,
}: {
    metas: Meta[];
    activeMeta: Meta | null;
    archetypes: ArchetypeStat[];
    selectedArchetypeId: string;
    minMatches: number;
    includeMirrors: boolean;
}) {
    return (
        <form className="selector selector--stack" method="get">
            <div className="selector__field">
                <p className="eyebrow">Meta</p>
                <label className="sr-only" htmlFor="meta_id">
                    Select meta
                </label>
                <select id="meta_id" name="meta_id" defaultValue={activeMeta?.id ?? ""}>
                    {metas.map((meta) => (
                        <option key={meta.id} value={meta.id}>
                            {meta.name}
                        </option>
                    ))}
                </select>
            </div>

            <div className="selector__field">
                <p className="eyebrow">Archetype</p>
                <label className="sr-only" htmlFor="archetype_id">
                    Filter by archetype
                </label>
                <select
                    id="archetype_id"
                    name="archetype_id"
                    defaultValue={selectedArchetypeId}
                >
                    <option value="">All archetypes</option>
                    {archetypes.map((archetype) => (
                        <option key={archetype.id} value={archetype.id}>
                            {archetype.name}
                        </option>
                    ))}
                </select>
            </div>

            <div className="selector__field">
                <p className="eyebrow">Minimum matches</p>
                <label className="sr-only" htmlFor="min_matches">
                    Minimum matches
                </label>
                <input
                    id="min_matches"
                    name="min_matches"
                    type="number"
                    min={1}
                    defaultValue={minMatches}
                />
            </div>

            <label className="selector__toggle">
                <input
                    type="checkbox"
                    name="include_mirrors"
                    value="true"
                    defaultChecked={includeMirrors}
                />
                Include mirror matchups
            </label>

            <button type="submit">Load matchups</button>
        </form>
    );
}

function MatchupTable({ stats }: { stats: MatchupStat[] }) {
    return (
        <div className="table-wrap">
            <table>
                <thead>
                    <tr>
                        <th>Archetype</th>
                        <th>Opponent</th>
                        <th>Record</th>
                        <th>Matches</th>
                        <th>Score rate</th>
                        <th>Win rate</th>
                    </tr>
                </thead>
                <tbody>
                    {stats.map((stat) => (
                        <tr key={`${stat.archetype.id}-${stat.opponent.id}`}>
                            <td>
                                <div className="table-title">{stat.archetype.name}</div>
                                <div className="muted tiny">{stat.archetype.slug}</div>
                            </td>
                            <td>
                                <div className="table-title">{stat.opponent.name}</div>
                                <div className="muted tiny">{stat.opponent.slug}</div>
                            </td>
                            <td>
                                {stat.wins}-{stat.losses}-{stat.ties}
                            </td>
                            <td>{stat.matches.toLocaleString()}</td>
                            <td>{formatPercent(stat.score_rate)}</td>
                            <td>{formatPercent(stat.win_rate)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

export default async function MatchupsPage({
    searchParams,
}: {
    searchParams: Promise<SearchParams>;
}) {
    const params = await searchParams;
    const metas = await getMetas().catch(() => [] as Meta[]);
    const activeMeta =
        metas.find((meta) => meta.id === params.meta_id) ??
        metas[0] ??
        null;

    const selectedArchetypeId = params.archetype_id ?? "";
    const parsedMinMatches = Number.parseInt(params.min_matches ?? "5", 10);
    const minMatches =
        Number.isFinite(parsedMinMatches) && parsedMinMatches > 0
            ? parsedMinMatches
            : 5;
    const includeMirrors = params.include_mirrors === "true";

    const [archetypes, matchupStats] = activeMeta
        ? await Promise.all([
              getArchetypeStats(activeMeta.id).catch(() => [] as ArchetypeStat[]),
              getMatchupStats({
                  metaId: activeMeta.id,
                  archetypeId: selectedArchetypeId || undefined,
                  minMatches,
                  includeMirrors,
              }).catch(() => [] as MatchupStat[]),
          ])
        : [[], []];

    const selectedArchetype =
        archetypes.find(
            (archetype) => String(archetype.id) === selectedArchetypeId,
        ) ?? null;

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <header className="hero card">
                    <div>
                        <p className="eyebrow">Matchup Explorer</p>
                        <h1>Head-to-head archetype matchup stats.</h1>
                        <p className="lede">
                            Compare directional matchup performance by meta and
                            archetype, including win rate and score rate from recorded
                            pairings.
                        </p>
                    </div>

                    <div className="hero__meta">
                        {activeMeta ? <span className="pill">{activeMeta.name}</span> : null}
                        {selectedArchetype ? (
                            <span className="pill pill--soft">{selectedArchetype.name}</span>
                        ) : null}
                    </div>
                </header>

                <section className="card section">
                    <div className="section__heading">
                        <div>
                            <p className="eyebrow">Filters</p>
                            <h2>Meta and matchup filters</h2>
                        </div>
                        <span className="muted">{matchupStats.length.toLocaleString()} rows</span>
                    </div>

                    {metas.length > 0 ? (
                        <MatchupFilters
                            metas={metas}
                            activeMeta={activeMeta}
                            archetypes={archetypes}
                            selectedArchetypeId={selectedArchetypeId}
                            minMatches={minMatches}
                            includeMirrors={includeMirrors}
                        />
                    ) : (
                        <EmptyState
                            title="No metas yet"
                            copy="Seed a meta before matchup stats can be loaded."
                        />
                    )}
                </section>

                <section className="card section section--spaced">
                    <div className="section__heading">
                        <div>
                            <p className="eyebrow">Matchups</p>
                            <h2>Archetype-vs-archetype results</h2>
                        </div>
                        <span className="muted">Directional from pairings data</span>
                    </div>

                    {matchupStats.length > 0 ? (
                        <MatchupTable stats={matchupStats} />
                    ) : (
                        <EmptyState
                            title="No matchup stats found"
                            copy="Try lowering the minimum match threshold or selecting a different meta."
                        />
                    )}
                </section>
            </div>
        </main>
    );
}
