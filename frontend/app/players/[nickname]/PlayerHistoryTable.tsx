"use client";

import Link from "next/link";
import Table from "@/components/table";
import type { PlayerHistoryEntry } from "@/lib/types";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so this column configuration lives
// here, in a client module, rather than inline in the (Server Component)
// player detail page.
function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
    }).format(new Date(value));
}

function formatPlacement(value: number) {
    return value === 0 ? "Dropped" : `#${value}`;
}

export default function PlayerHistoryTable({
    nickname,
    history,
}: {
    nickname: string;
    history: PlayerHistoryEntry[];
}) {
    return (
        <Table
            columns={[
                {
                    key: "placement",
                    label: "Placement",
                    render: (r: PlayerHistoryEntry) =>
                        formatPlacement(r.placement),
                    sortValue: (r: PlayerHistoryEntry) =>
                        r.placement === 0 ? Number.MAX_SAFE_INTEGER : r.placement,
                },
                {
                    key: "event",
                    label: "Event",
                    render: (r: PlayerHistoryEntry) => (
                        <Link
                            className="table-link"
                            href={`/tournaments/${r.tournament_id}`}
                        >
                            <div className="table-title">{r.event_name}</div>
                        </Link>
                    ),
                    sortValue: (r: PlayerHistoryEntry) => r.event_name,
                },
                {
                    key: "date",
                    label: "Date",
                    render: (r: PlayerHistoryEntry) => formatDate(r.date),
                    sortValue: (r: PlayerHistoryEntry) => r.date,
                },
                {
                    key: "players",
                    label: "Players",
                    render: (r: PlayerHistoryEntry) => r.players,
                    sortValue: (r: PlayerHistoryEntry) => r.players,
                },
                {
                    key: "archetype",
                    label: "Archetype",
                    render: (r: PlayerHistoryEntry) =>
                        r.archetype_name ?? (
                            <span className="muted tiny">Unknown</span>
                        ),
                    sortValue: (r: PlayerHistoryEntry) => r.archetype_name,
                },
                {
                    key: "decklist",
                    label: "Decklist",
                    sortable: false,
                    render: (r: PlayerHistoryEntry) =>
                        r.decklist_id !== null ? (
                            <Link
                                className="button"
                                href={`/players/${encodeURIComponent(nickname)}/decklist/${r.decklist_id}`}
                            >
                                View decklist
                            </Link>
                        ) : (
                            <span className="muted tiny">—</span>
                        ),
                },
            ]}
            rows={history}
        />
    );
}
