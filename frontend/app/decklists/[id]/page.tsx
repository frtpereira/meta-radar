import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";
import Table from "@/components/table";

import type { CardStat, MatchupStat } from "@/lib/types";
import {
    getArchetypeCardStats,
    getArchetypeDetail,
    getArchetypeStats,
    getMatchupStats,
} from "@/lib/api";

// ─── helpers ──────────────────────────────────────────────────────────────────

function formatPercent(value: number | null) {
    if (value === null) return "—";
    return `${Math.round(value * 1000) / 10}%`;
}

function formatStanding(value: number | null) {
    if (value === null) return "—";
    return `#${Math.round(value)}`;
}

// ─── sub-components ───────────────────────────────────────────────────────────

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

/** Thin horizontal progress bar, gold→coral gradient */
function PresenceBar({ value }: { value: number }) {
    return (
        <div
            style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                minWidth: 120,
            }}
        >
            <div
                style={{
                    flex: 1,
                    height: 5,
                    background: "var(--overlay-med)",
                    borderRadius: 3,
                    overflow: "hidden",
                }}
            >
                <div
                    style={{
                        width: `${Math.round(value * 100)}%`,
                        height: "100%",
                        background:
                            "linear-gradient(90deg, var(--accent), var(--accent-2))",
                        borderRadius: 3,
                    }}
                />
            </div>
            <span
                className="tiny"
                style={{
                    color:
                        value >= 0.9
                            ? "var(--success)"
                            : value >= 0.7
                              ? "var(--accent)"
                              : "var(--muted)",
                    fontWeight: 600,
                    minWidth: 38,
                    textAlign: "right",
                }}
            >
                {Math.round(value * 100)}%
            </span>
        </div>
    );
}

/** Compact copy-count distribution: ×4 85%  ×3 15% */
function CountDist({
    dist,
    totalDecklists,
}: {
    dist: Record<string, number>;
    totalDecklists: number;
}) {
    const entries = Object.entries(dist)
        .map(([cnt, frac]) => ({ cnt: Number(cnt), frac }))
        .sort((a, b) => b.frac - a.frac);

    if (entries.length === 0) return <span className="muted">—</span>;

    return (
        <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            {entries.map(({ cnt, frac }) => (
                <span
                    key={cnt}
                    style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 3,
                        padding: "2px 8px",
                        borderRadius: 999,
                        border: "1px solid var(--line)",
                        background: "var(--overlay-soft)",
                        fontSize: "0.76rem",
                        whiteSpace: "nowrap",
                    }}
                >
                    <span style={{ color: "var(--accent)", fontWeight: 700 }}>
                        ×{cnt}
                    </span>
                    <span className="muted">{Math.round(frac * 100)}%</span>
                </span>
            ))}
        </div>
    );
}

/** One category block inside the skeleton section */
function SkeletonCategory({
    label,
    cards,
    totalDecklists,
}: {
    label: string;
    cards: CardStat[];
    totalDecklists: number;
}) {
    if (cards.length === 0) return null;

    const columns = [
        {
            key: "count",
            label: "Count",
            render: (c: CardStat) => (
                <span
                    style={{
                        display: "inline-flex",
                        alignItems: "center",
                        justifyContent: "center",
                        width: 28,
                        height: 28,
                        borderRadius: "50%",
                        background: "rgba(255,209,102,0.12)",
                        border: "1px solid rgba(255,209,102,0.3)",
                        color: "var(--accent)",
                        fontWeight: 700,
                        fontSize: "0.85rem",
                        fontFamily: "Georgia, serif",
                    }}
                >
                    {c.modal_count}
                </span>
            ),
        },
        {
            key: "card",
            label: "Card",
            render: (c: CardStat) => (
                <div>
                    <div className="table-title">{c.name}</div>
                    {c.set ? (
                        <div className="muted tiny">
                            {c.set}
                            {c.number ? ` ${c.number}` : ""}
                        </div>
                    ) : null}
                </div>
            ),
        },
        {
            key: "presence",
            label: "Usage",
            render: (c: CardStat) => <PresenceBar value={c.presence} />,
        },
        {
            key: "dist",
            label: "Count distribution",
            render: (c: CardStat) => (
                <CountDist
                    dist={c.count_distribution}
                    totalDecklists={totalDecklists}
                />
            ),
        },
    ];

    return (
        <div style={{ marginTop: 24 }}>
            <p
                className="eyebrow"
                style={{
                    marginBottom: 10,
                    paddingBottom: 8,
                    borderBottom: "1px solid var(--line)",
                }}
            >
                {label}
            </p>
            <Table columns={columns} rows={cards} />
        </div>
    );
}

/** Best / Worst matchup mini-table */
function MatchupMiniTable({
    stats,
    archetypeId,
    label,
    colorFn,
}: {
    stats: MatchupStat[];
    archetypeId: number;
    label: string;
    colorFn: (wr: number | null) => string;
}) {
    if (stats.length === 0) {
        return (
            <EmptyState
                title={`No ${label.toLowerCase()} data`}
                copy="Run the pairings pipeline to populate matchup stats."
            />
        );
    }

    const columns = [
        {
            key: "opponent",
            label: "Opponent",
            render: (s: MatchupStat) => {
                const opp =
                    String(s.archetype.id) === String(archetypeId)
                        ? s.opponent
                        : s.archetype;
                return (
                    <div>
                        <div className="table-title">{opp.name}</div>
                        <div className="muted tiny">{opp.slug}</div>
                    </div>
                );
            },
        },
        {
            key: "record",
            label: "Record",
            render: (s: MatchupStat) => {
                const weAreArchetype =
                    String(s.archetype.id) === String(archetypeId);
                const w = weAreArchetype ? s.wins : s.losses;
                const l = weAreArchetype ? s.losses : s.wins;
                return (
                    <span className="muted tiny">
                        {w}–{l}–{s.ties}
                    </span>
                );
            },
        },
        {
            key: "win_rate",
            label: "Win rate",
            render: (s: MatchupStat) => {
                const weAreArchetype =
                    String(s.archetype.id) === String(archetypeId);
                const w = weAreArchetype ? s.wins : s.losses;
                const l = weAreArchetype ? s.losses : s.wins;
                const total = w + l + s.ties;
                const wr = total > 0 ? w / total : null;
                return (
                    <span
                        style={{
                            color: colorFn(wr),
                            fontWeight: 700,
                            fontFamily: "Georgia, serif",
                        }}
                    >
                        {formatPercent(wr)}
                    </span>
                );
            },
        },
        {
            key: "matches",
            label: "Matches",
            render: (s: MatchupStat) => (
                <span className="muted tiny">{s.matches.toLocaleString()}</span>
            ),
        },
    ];

    return <Table columns={columns} rows={stats} />;
}

// ─── page ─────────────────────────────────────────────────────────────────────

type PageParams = { id: string };
type SearchParams = { meta_id?: string };

export default async function DecklistDetailPage({
    params,
    searchParams,
}: {
    params: Promise<PageParams>;
    searchParams: Promise<SearchParams>;
}) {
    const { id } = await params;
    const { meta_id } = await searchParams;

    // Archetype detail (name, slug, core_cards, threshold)
    const archetype = await getArchetypeDetail(id).catch((err: unknown) => {
        if (
            err instanceof Error &&
            err.message.startsWith("Request failed: 404")
        ) {
            notFound();
        }
        throw err;
    });

    // Resolve meta: prefer explicit meta_id param, fall back to archetype's own meta
    const resolvedMetaId = meta_id ?? archetype.meta_id;

    // Fetch the rest in parallel — gracefully degrade on failure
    const [cardStats, archetypeStats, matchupPage] = await Promise.all([
        getArchetypeCardStats(id).catch(
            () => [] as import("@/lib/types").CardStat[],
        ),
        getArchetypeStats(resolvedMetaId).catch(() => []),
        getMatchupStats({
            metaId: resolvedMetaId,
            archetypeId: id,
            minMatches: 1,
            includeMirrors: false,
            pageSize: 100,
        }).catch(() => ({
            items: [] as MatchupStat[],
            total: 0,
            page: 1,
            page_size: 100,
        })),
    ]);

    const totalDecklists =
        cardStats.length > 0 ? cardStats[0].total_decklists : 0;

    const thisStat = archetypeStats.find((s) => String(s.id) === id);

    // Split cards into skeleton (core) and optional
    const MIN_OPTIONAL_CARD_PRESENCE = 0.1;
    const skeletonCards = cardStats.filter((c) => c.is_core);
    const optionalCards = cardStats.filter(
        (c) => !c.is_core && c.presence >= MIN_OPTIONAL_CARD_PRESENCE,
    );

    // Group skeleton by category
    const byCategory = (cat: string) =>
        skeletonCards
            .filter((c) => c.category === cat)
            .sort((a, b) => b.presence - a.presence);

    const pokemonCards = byCategory("pokemon");
    const trainerCards = byCategory("trainer");
    const energyCards = byCategory("energy");

    // Derive per-archetype win rates from matchup stats, then rank
    const matchupsWithWinRate = matchupPage.items.map((s) => {
        const weAreArchetype = String(s.archetype.id) === id;
        const w = weAreArchetype ? s.wins : s.losses;
        const l = weAreArchetype ? s.losses : s.wins;
        const total = w + l + s.ties;
        const wr = total > 0 ? w / total : null;
        return { stat: s, winRate: wr, total };
    });

    const MIN_MATCHUP_SAMPLE_SIZE = 10;

    const ranked = [...matchupsWithWinRate]
        .filter((m) => m.winRate !== null && m.total >= MIN_MATCHUP_SAMPLE_SIZE)
        .sort((a, b) => (b.winRate ?? 0) - (a.winRate ?? 0));

    const bestAgainst = ranked.slice(0, 5).map((m) => m.stat);
    const worstAgainst = ranked
        .slice(-5)
        .reverse()
        .map((m) => m.stat);

    const goodColor = (wr: number | null) =>
        wr === null ? "var(--muted)" : wr >= 0.55 ? "var(--success)" : "var(--accent)";
    const badColor = (wr: number | null) =>
        wr === null
            ? "var(--muted)"
            : wr < 0.4
              ? "var(--accent-2)"
              : "var(--muted)";

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                {/* breadcrumb */}
                <div style={{ marginBottom: 16 }}>
                    <Link
                        href={`/decklists${resolvedMetaId ? `?meta_id=${resolvedMetaId}` : ""}`}
                        className="button"
                        style={{ display: "inline-flex" }}
                    >
                        ← All archetypes
                    </Link>
                </div>

                <Hero
                    eyebrow="Meta Radar — Decklist"
                    title={archetype.name}
                    lede={`Deck skeleton, tech cards, and matchup breakdown for ${archetype.name} in this meta.`}
                    meta={
                        <>
                            <span className="pill">
                                {totalDecklists.toLocaleString()} decklists
                            </span>
                            {archetype.core_threshold !== null ? (
                                <span className="pill pill--soft">
                                    Skeleton threshold:{" "}
                                    {Math.round(archetype.core_threshold * 100)}
                                    %
                                </span>
                            ) : null}
                            {thisStat?.win_rate !== undefined &&
                            thisStat.win_rate !== null ? (
                                <span className="pill pill--soft">
                                    Win rate: {formatPercent(thisStat.win_rate)}
                                </span>
                            ) : null}
                            {thisStat?.avg_standing !== undefined &&
                            thisStat.avg_standing !== null ? (
                                <span className="pill pill--soft">
                                    Avg standing:{" "}
                                    {formatStanding(thisStat.avg_standing)}
                                </span>
                            ) : null}
                        </>
                    }
                />

                {/* ── Deck Skeleton ─────────────────────────────────────── */}
                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Core cards</p>
                            <h2>Deck Skeleton</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {skeletonCards.length} cards · present in ≥
                            {archetype.core_threshold !== null
                                ? Math.round(archetype.core_threshold * 100)
                                : 70}
                            % of lists
                        </span>
                    }
                >
                    {skeletonCards.length === 0 ? (
                        <EmptyState
                            title="No skeleton data yet"
                            copy="Run the clustering pipeline (cmd/cluster) for this meta to compute core cards."
                        />
                    ) : (
                        <>
                            <SkeletonCategory
                                label="Pokémon"
                                cards={pokemonCards}
                                totalDecklists={totalDecklists}
                            />
                            <SkeletonCategory
                                label="Trainer"
                                cards={trainerCards}
                                totalDecklists={totalDecklists}
                            />
                            <SkeletonCategory
                                label="Energy"
                                cards={energyCards}
                                totalDecklists={totalDecklists}
                            />
                        </>
                    )}
                </Card>

                {/* ── Optional Cards ────────────────────────────────────── */}
                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Tech slots</p>
                            <h2>Optional Cards</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {optionalCards.length} cards below skeleton
                            threshold
                        </span>
                    }
                >
                    {optionalCards.length === 0 ? (
                        <EmptyState
                            title="No optional card data"
                            copy="Run the clustering pipeline to separate core cards from tech choices."
                        />
                    ) : (
                        <Table
                            columns={[
                                {
                                    key: "card",
                                    label: "Card",
                                    render: (c: CardStat) => (
                                        <div>
                                            <div className="table-title">
                                                {c.name}
                                            </div>
                                            {c.set ? (
                                                <div className="muted tiny">
                                                    {c.set}
                                                    {c.number
                                                        ? ` ${c.number}`
                                                        : ""}
                                                </div>
                                            ) : null}
                                        </div>
                                    ),
                                },
                                {
                                    key: "category",
                                    label: "Category",
                                    render: (c: CardStat) => (
                                        <span className="badge">
                                            {c.category === "pokemon"
                                                ? "Pokémon"
                                                : c.category === "trainer"
                                                  ? "Trainer"
                                                  : c.category === "energy"
                                                    ? "Energy"
                                                    : c.category}
                                        </span>
                                    ),
                                },
                                {
                                    key: "usage",
                                    label: "Usage",
                                    render: (c: CardStat) => (
                                        <PresenceBar value={c.presence} />
                                    ),
                                },
                                {
                                    key: "typical",
                                    label: "Typical count",
                                    render: (c: CardStat) => (
                                        <span
                                            style={{
                                                color: "var(--accent)",
                                                fontWeight: 700,
                                                fontFamily: "Georgia, serif",
                                            }}
                                        >
                                            ×{c.modal_count}
                                        </span>
                                    ),
                                },
                                {
                                    key: "dist",
                                    label: "Count distribution",
                                    render: (c: CardStat) => (
                                        <CountDist
                                            dist={c.count_distribution}
                                            totalDecklists={totalDecklists}
                                        />
                                    ),
                                },
                            ]}
                            rows={optionalCards}
                        />
                    )}
                </Card>

                {/* ── Matchups ──────────────────────────────────────────── */}
                <section className="grid grid--two section--spaced">
                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Matchups</p>
                                <h2>Best against</h2>
                            </>
                        }
                        headingMeta={
                            <span
                                style={{
                                    color: "var(--success)",
                                    fontSize: "0.84rem",
                                    fontWeight: 600,
                                }}
                            >
                                Top 5 favourable
                            </span>
                        }
                    >
                        <MatchupMiniTable
                            stats={bestAgainst}
                            archetypeId={archetype.id}
                            label="Best against"
                            colorFn={goodColor}
                        />
                    </Card>

                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Matchups</p>
                                <h2>Worst against</h2>
                            </>
                        }
                        headingMeta={
                            <span
                                style={{
                                    color: "var(--accent-2)",
                                    fontSize: "0.84rem",
                                    fontWeight: 600,
                                }}
                            >
                                Top 5 unfavourable
                            </span>
                        }
                    >
                        <MatchupMiniTable
                            stats={worstAgainst}
                            archetypeId={archetype.id}
                            label="Worst against"
                            colorFn={badColor}
                        />
                    </Card>
                </section>

                {/* full matchups link */}
                <div
                    style={{
                        marginTop: 16,
                        display: "flex",
                        justifyContent: "flex-end",
                    }}
                >
                    <Link
                        href={`/matchups?meta_id=${resolvedMetaId}&archetype_id=${id}`}
                        className="button"
                    >
                        View all matchups →
                    </Link>
                </div>
            </div>
        </main>
    );
}
