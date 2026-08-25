import type { ArchetypeStat, MatchupStat, Meta } from "@/lib/types";

import { getArchetypeStats, getMatchupStats, getMetas } from "@/lib/api";
import Pagination from "@/components/pagination";
import Hero from "@/components/hero";
import Table from "@/components/table";
import Card from "@/components/card";
import InfoTooltip from "@/components/info-tooltip";

type SearchParams = {
    meta_id?: string;
    archetype_id?: string;
    min_matches?: string;
    page?: string;
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
}: {
    metas: Meta[];
    activeMeta: Meta | null;
    archetypes: ArchetypeStat[];
    selectedArchetypeId: string;
    minMatches: number;
}) {
    return (
        <form className="selector selector--stack" method="get">
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

            <button type="submit">Load matchups</button>
        </form>
    );
}

function MatchupTable({
    stats,
    selectedArchetypeId,
}: {
    stats: MatchupStat[];
    selectedArchetypeId: string;
}) {
    const columns = [
        {
            key: "primary",
            label: "Archetype",
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                const primary = selectedIsArchetype
                    ? stat.archetype
                    : stat.opponent;
                return <div className="table-title">{primary.name}</div>;
            },
            sortValue: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype
                    ? stat.archetype.name
                    : stat.opponent.name;
            },
        },
        {
            key: "secondary",
            label: "Opponent",
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                const secondary = selectedIsArchetype
                    ? stat.opponent
                    : stat.archetype;
                return <div className="table-title">{secondary.name}</div>;
            },
            sortValue: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype
                    ? stat.opponent.name
                    : stat.archetype.name;
            },
        },
        {
            key: "record",
            label: "Record",
            // The displayed W-L-T string depends on perspective/mirror
            // status and isn't a single meaningful sort key, so leave this
            // column unsortable.
            sortable: false,
            render: (stat: MatchupStat) => {
                // W-L is meaningless for a mirror: both sides are the same
                // archetype, so wins and losses are equal by definition.
                const isMirror = stat.archetype.id === stat.opponent.id;
                if (isMirror) {
                    return `${stat.ties} ties`;
                }

                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                const displayedWins = selectedIsArchetype
                    ? stat.wins
                    : stat.losses;
                const displayedLosses = selectedIsArchetype
                    ? stat.losses
                    : stat.wins;
                const displayedTies = stat.ties;
                return `${displayedWins}-${displayedLosses}-${displayedTies}`;
            },
        },
        {
            key: "matches",
            label: "Matches",
            render: (stat: MatchupStat) => stat.matches.toLocaleString(),
            sortValue: (stat: MatchupStat) => stat.matches,
        },
        {
            key: "score_rate",
            label: (
                <>
                    Score rate
                    <InfoTooltip text="Share of possible match points earned in this matchup, from the highlighted archetype's perspective: (wins + 0.5 × ties) ÷ matches played. Unlike win rate, ties count as half a win instead of being excluded." />
                </>
            ),
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                // score_rate is stored from `stat.archetype`'s perspective
                // and is always a fraction in [0, 1]. When the row is being
                // displayed from the opponent's perspective, the correct
                // complement is `1 - score_rate` (matches the win_rate swap
                // above), not `-score_rate` — negating it produced bogus
                // negative percentages.
                const displayedScoreRate =
                    stat.score_rate === null
                        ? null
                        : selectedIsArchetype
                          ? stat.score_rate
                          : 1 - stat.score_rate;
                return formatPercent(displayedScoreRate);
            },
            sortValue: (stat: MatchupStat) => {
                if (stat.score_rate === null) {
                    return null;
                }
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype
                    ? stat.score_rate
                    : 1 - stat.score_rate;
            },
        },
        {
            key: "win_rate",
            label: "Win rate",
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                // win_rate is null for mirrors (see backend) -- trust that
                // instead of recomputing from wins/losses, which are equal
                // for a mirror and would otherwise render a bogus 50%.
                const displayedWinRate =
                    stat.win_rate === null
                        ? null
                        : selectedIsArchetype
                          ? stat.win_rate
                          : 1 - stat.win_rate;
                return formatPercent(displayedWinRate);
            },
            sortValue: (stat: MatchupStat) => {
                if (stat.win_rate === null) {
                    return null;
                }
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype ? stat.win_rate : 1 - stat.win_rate;
            },
        },
    ];

    return <Table columns={columns} rows={stats} />;
}

export default async function MatchupsPage({
    searchParams,
}: {
    searchParams: Promise<SearchParams>;
}) {
    const params = await searchParams;
    const metas = await getMetas().catch(() => [] as Meta[]);
    const activeMeta =
        metas.find((meta) => meta.id === params.meta_id) ?? metas[0] ?? null;

    const selectedArchetypeId = params.archetype_id ?? "";
    const parsedMinMatches = Number.parseInt(params.min_matches ?? "20", 10);
    const minMatches =
        Number.isFinite(parsedMinMatches) && parsedMinMatches > 0
            ? parsedMinMatches
            : 20;
    const page = Number.parseInt(params.page ?? "1", 10) || 1;
    const pageSize = 20;

    const [archetypes, matchupPage] = activeMeta
        ? await Promise.all([
              getArchetypeStats(activeMeta.id).catch(
                  () => [] as ArchetypeStat[],
              ),
              getMatchupStats({
                  metaId: activeMeta.id,
                  archetypeId: selectedArchetypeId || undefined,
                  minMatches,
                  page,
                  pageSize,
              }).catch(() => ({
                  total: 0,
                  page,
                  page_size: pageSize,
                  items: [],
              })),
          ])
        : [[], { total: 0, page, page_size: pageSize, items: [] }];

    const selectedArchetype =
        archetypes.find(
            (archetype) => String(archetype.id) === selectedArchetypeId,
        ) ?? null;

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="Meta Radar - Matchups"
                    title="Matchup Analysis"
                    lede="Compare directional matchup performance by meta and
                        archetype, including win rate and score rate from
                        recorded pairings."
                    meta={
                        <>
                            {activeMeta ? (
                                <span className="pill">{activeMeta.name}</span>
                            ) : null}
                            {selectedArchetype ? (
                                <span className="pill pill--soft">
                                    {selectedArchetype.name}
                                </span>
                            ) : null}
                        </>
                    }
                />

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Filters</p>
                            <h2>Meta and Deck Filters</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {matchupPage.total.toLocaleString()} rows
                        </span>
                    }
                >
                    {metas.length > 0 ? (
                        <MatchupFilters
                            metas={metas}
                            activeMeta={activeMeta}
                            archetypes={archetypes}
                            selectedArchetypeId={selectedArchetypeId}
                            minMatches={minMatches}
                        />
                    ) : (
                        <EmptyState
                            title="No metas yet"
                            copy="Seed a meta before matchup stats can be loaded."
                        />
                    )}
                </Card>

                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Matchups</p>
                            <h2>Head-to-Head Results</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            Directional from pairings data
                        </span>
                    }
                >
                    {matchupPage.items.length > 0 ? (
                        <>
                            <MatchupTable
                                stats={matchupPage.items}
                                selectedArchetypeId={selectedArchetypeId}
                            />
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
                                        Page {matchupPage.page} of{" "}
                                        {Math.max(
                                            1,
                                            Math.ceil(
                                                matchupPage.total /
                                                    matchupPage.page_size,
                                            ),
                                        )}
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
                                        page={matchupPage.page}
                                        totalPages={Math.max(
                                            1,
                                            Math.ceil(
                                                matchupPage.total /
                                                    matchupPage.page_size,
                                            ),
                                        )}
                                    />
                                </div>
                            </div>
                        </>
                    ) : (
                        <EmptyState
                            title="No matchup stats found"
                            copy="Try lowering the minimum match threshold or selecting a different meta."
                        />
                    )}
                </Card>
            </div>
        </main>
    );
}
