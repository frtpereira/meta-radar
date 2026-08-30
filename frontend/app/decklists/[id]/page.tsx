import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";

import type { MatchupStat } from "@/lib/types";
import {
    getArchetypeCardStats,
    getArchetypeDetail,
    getArchetypeStats,
    getMatchupStats,
} from "@/lib/api";
import {
    MatchupMiniTable,
    OptionalCardsTable,
    SkeletonCategory,
} from "./DecklistTables";

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
    const [cardStats, archetypeStats, matchupStats] = await Promise.all([
        getArchetypeCardStats(id).catch(
            () => [] as import("@/lib/types").CardStat[],
        ),
        getArchetypeStats(resolvedMetaId).catch(() => []),
        getMatchupStats({
            metaId: resolvedMetaId,
            archetypeId: id,
            minMatches: 1,
            includeMirrors: false,
        }).catch(() => [] as MatchupStat[]),
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
    const matchupsWithWinRate = matchupStats.map((s) => {
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
                    eyebrow="Meta Radar — Archetype"
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
                            <p className="eyebrow">Decklist</p>
                            <h2>Core/Skeleton</h2>
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
                        <OptionalCardsTable cards={optionalCards} />
                    )}
                </Card>

                {/* ── Matchups ──────────────────────────────────────────── */}
                <section className="grid grid--two section--spaced">
                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Matchups</p>
                                <h2>Best Against</h2>
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
                            variant="good"
                            metaId={resolvedMetaId}
                        />
                    </Card>

                    <Card
                        heading={
                            <>
                                <p className="eyebrow">Matchups</p>
                                <h2>Worst Against</h2>
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
                            variant="bad"
                            metaId={resolvedMetaId}
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
