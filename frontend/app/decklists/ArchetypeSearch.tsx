"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import type { ArchetypeStat } from "@/lib/types";
import Table from "@/components/table";
import InfoTooltip from "@/components/info-tooltip";

const PAGE_SIZE = 20;

// Client-side pagination window, mirroring components/pagination.tsx's
// prev/next + numbered-window layout, but driven by local state instead
// of the URL: the archetype list here is filtered entirely in the
// browser (see the note on ArchetypeSearch below), so there's no page
// to navigate to on the server -- flipping pages just re-slices the
// already-filtered array.
function ArchetypePagination({
    page,
    totalPages,
    onPageChange,
}: {
    page: number;
    totalPages: number;
    onPageChange: (page: number) => void;
}) {
    const windowSize = 2; // show current +/- 2
    const pages: number[] = [];
    for (
        let i = Math.max(1, page - windowSize);
        i <= Math.min(totalPages, page + windowSize);
        i++
    ) {
        pages.push(i);
    }

    return (
        <div className="pagination" style={{ marginTop: 12 }}>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <button
                    className="button"
                    onClick={() => onPageChange(Math.max(1, page - 1))}
                    disabled={page <= 1}
                >
                    Prev
                </button>

                {pages.map((p) => (
                    <button
                        key={p}
                        className={`button ${p === page ? "button--active" : ""}`}
                        onClick={() => onPageChange(p)}
                        aria-current={p === page}
                    >
                        {p}
                    </button>
                ))}

                <button
                    className="button"
                    onClick={() => onPageChange(Math.min(totalPages, page + 1))}
                    disabled={page >= totalPages}
                >
                    Next
                </button>
            </div>
        </div>
    );
}

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
    ];

    return <Table columns={columns} rows={archetypes} />;
}

// Client-side, instant filtering: the archetype list for a meta is small
// (one page of data, not paginated by the API), so filtering what's
// already loaded avoids a server round-trip and sidesteps the pagination
// concerns a server-side search would raise on other list endpoints.
//
// Pagination (below) follows the same shape as the /matchups page, but
// paginates over the already-filtered, in-browser array instead of a
// server-paginated response -- the API isn't paginated here, so there's
// nothing to fetch per page. Search stays fully functional: it filters
// the complete archetype list before pagination ever slices it, so a
// match on page 3 is still found even if you're currently viewing page 1.
export default function ArchetypeSearch({
    archetypes,
    metaId,
}: {
    archetypes: ArchetypeStat[];
    metaId: string;
}) {
    const [query, setQuery] = useState("");
    const [page, setPage] = useState(1);

    const filtered = useMemo(() => {
        const trimmed = query.trim().toLowerCase();
        if (!trimmed) return archetypes;
        return archetypes.filter((a) => a.name.toLowerCase().includes(trimmed));
    }, [archetypes, query]);

    const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
    // Clamp instead of resetting state directly during render: keeps the
    // displayed page in range if a filter shrinks the result set without
    // fighting the explicit page=1 reset in handleQueryChange below.
    const safePage = Math.min(page, totalPages);
    const paged = filtered.slice(
        (safePage - 1) * PAGE_SIZE,
        safePage * PAGE_SIZE,
    );

    function handleQueryChange(value: string) {
        setQuery(value);
        setPage(1);
    }

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
                        onChange={(e) => handleQueryChange(e.target.value)}
                        style={{ width: "100%", minWidth: "auto" }}
                    />
                </div>
                <span className="muted">
                    {filtered.length.toLocaleString()} archetypes
                </span>
            </div>

            {filtered.length > 0 ? (
                <>
                    <ArchetypesTable archetypes={paged} metaId={metaId} />
                    {totalPages > 1 ? (
                        <div
                            style={{
                                display: "flex",
                                flexDirection: "column",
                                gap: "4px",
                            }}
                        >
                            <span className="muted">
                                Page {safePage} of {totalPages}
                            </span>
                            <ArchetypePagination
                                page={safePage}
                                totalPages={totalPages}
                                onPageChange={setPage}
                            />
                        </div>
                    ) : null}
                </>
            ) : (
                <EmptyState
                    title="No matching archetypes"
                    copy={`No archetypes match "${query}".`}
                />
            )}
        </>
    );
}
