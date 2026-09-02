"use client";

import Link from "next/link";
import Table from "@/components/table";
import type { CardStat, MatchupStat } from "@/lib/types";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so the column configuration for
// this page's tables lives here, in a client module, rather than inline in
// the (Server Component) archetype detail page.

function formatPercent(value: number | null) {
    if (value === null) return "—";
    return `${Math.round(value * 1000) / 10}%`;
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
function CountDist({ dist }: { dist: Record<string, number> }) {
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
export function SkeletonCategory({
    label,
    cards,
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
            sortValue: (c: CardStat) => c.modal_count,
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
            sortValue: (c: CardStat) => c.name,
        },
        {
            key: "presence",
            label: "Usage",
            render: (c: CardStat) => <PresenceBar value={c.presence} />,
        },
        {
            key: "dist",
            label: "Count distribution",
            render: (c: CardStat) => <CountDist dist={c.count_distribution} />,
            sortable: false,
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

export function OptionalCardsTable({ cards }: { cards: CardStat[] }) {
    return (
        <Table
            columns={[
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
                    sortValue: (c: CardStat) => c.name,
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
                    render: (c: CardStat) => <PresenceBar value={c.presence} />,
                    sortValue: (c: CardStat) => c.presence,
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
                    sortValue: (c: CardStat) => c.modal_count,
                },
                {
                    key: "dist",
                    label: "Count distribution",
                    render: (c: CardStat) => (
                        <CountDist dist={c.count_distribution} />
                    ),
                    sortable: false,
                },
            ]}
            rows={cards}
        />
    );
}

/** Best / Worst matchup mini-table */
export function MatchupMiniTable({
    stats,
    archetypeId,
    label,
    variant,
    metaId,
}: {
    stats: MatchupStat[];
    archetypeId: number;
    label: string;
    variant: "good" | "bad";
    metaId?: string;
}) {
    const colorFn = (wr: number | null) => {
        if (wr === null) {
            return "var(--muted)";
        }
        if (variant === "good") {
            return wr >= 0.55 ? "var(--success)" : "var(--accent)";
        }
        return wr < 0.4 ? "var(--accent-2)" : "var(--muted)";
    };

    if (stats.length === 0) {
        return (
            <div className="empty-state">
                <h3>No {label.toLowerCase()} data</h3>
                <p>Run the pairings pipeline to populate matchup stats.</p>
            </div>
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
                    <Link
                        className="table-link"
                        href={`/archetypes/${opp.id}${metaId ? `?meta_id=${metaId}` : ""}`}
                    >
                        <div className="table-title">{opp.name}</div>
                    </Link>
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

    // Best/worst against are fixed top-5 previews -- too short to bother
    // sorting.
    return <Table columns={columns} rows={stats} sortable={false} />;
}
