"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import type { ArchetypeStat } from "@/lib/types";
import Table from "@/components/table";
import InfoTooltip from "@/components/info-tooltip";

function formatPercent(value: number | null) {
    if (value === null) return "—";
    return `${Math.round(value * 1000) / 10}%`;
}

function formatStanding(value: number | null) {
    if (value === null) return "—";
    return `#${Math.round(value)}`;
}

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

function WinRateBadge({ value }: { value: number | null }) {
    if (value === null) return <span className="muted">—</span>;
    const pct = Math.round(value * 1000) / 10;
    const color =
        pct >= 55
            ? "var(--success)"
            : pct >= 48
              ? "var(--accent)"
              : "var(--accent-2)";
    return <span style={{ color, fontWeight: 600 }}>{pct}%</span>;
}

function ArchetypesTable({
    archetypes,
    metaId,
}: {
    archetypes: ArchetypeStat[];
    metaId: string;
}) {
    const columns = [
        {
            key: "archetype",
            label: "Archetype",
            render: (s: ArchetypeStat) => (
                <Link
                    className="table-link"
                    href={`/decklists/${s.id}?meta_id=${metaId}`}
                >
                    <div className="table-title">{s.name}</div>
                </Link>
            ),
        },
        {
            key: "decks",
            label: "Decklists",
            render: (s: ArchetypeStat) => s.deck_count.toLocaleString(),
        },
        {
            key: "win_rate",
            label: "Win rate",
            render: (s: ArchetypeStat) => <WinRateBadge value={s.win_rate} />,
        },
        {
            key: "score_rate",
            label: (
                <>
                    Score rate
                    <InfoTooltip text="Share of possible match points earned: (wins + 0.5 × ties) ÷ matches played. Unlike win rate, ties count as half a win instead of being excluded." />
                </>
            ),
            render: (s: ArchetypeStat) => formatPercent(s.score_rate),
        },
        {
            key: "avg_standing",
            label: "Avg standing",
            render: (s: ArchetypeStat) => formatStanding(s.avg_standing),
        },
        {
            key: "record",
            label: "Record",
            render: (s: ArchetypeStat) => `${s.wins}–${s.losses}–${s.ties}`,
        },
        {
            key: "action",
            label: "",
            render: (s: ArchetypeStat) => (
                <Link
                    href={`/decklists/${s.id}?meta_id=${metaId}`}
                    className="button"
                    style={{ whiteSpace: "nowrap" }}
                >
                    View deck →
                </Link>
            ),
        },
    ];

    return <Table columns={columns} rows={archetypes} />;
}

// Client-side, instant filtering: the archetype list for a meta is small
// (one page of data, not paginated by the API), so filtering what's
// already loaded avoids a server round-trip and sidesteps the pagination
// concerns a server-side search would raise on other list endpoints.
export default function ArchetypeSearch({
    archetypes,
    metaId,
}: {
    archetypes: ArchetypeStat[];
    metaId: string;
}) {
    const [query, setQuery] = useState("");

    const filtered = useMemo(() => {
        const trimmed = query.trim().toLowerCase();
        if (!trimmed) return archetypes;
        return archetypes.filter((a) => a.name.toLowerCase().includes(trimmed));
    }, [archetypes, query]);

    return (
        <>
            <div className="selector">
                <div className="selector__field" style={{ flex: 1 }}>
                    <label className="sr-only" htmlFor="archetype_search">
                        Search archetypes
                    </label>
                    <input
                        id="archetype_search"
                        type="search"
                        placeholder="Search archetypes…"
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        style={{ width: "100%", minWidth: "auto" }}
                    />
                </div>
                <span className="muted">
                    {filtered.length.toLocaleString()} archetypes
                </span>
            </div>

            {filtered.length > 0 ? (
                <ArchetypesTable archetypes={filtered} metaId={metaId} />
            ) : (
                <EmptyState
                    title="No matching archetypes"
                    copy={`No archetypes match "${query}".`}
                />
            )}
        </>
    );
}
