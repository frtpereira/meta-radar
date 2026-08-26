"use client";

import Link from "next/link";
import Table from "@/components/table";
import type { Tournament } from "@/lib/types";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so this column configuration lives
// here, in a client module, rather than inline in the (Server Component)
// tournaments page.
function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
    }).format(new Date(value));
}

export default function TournamentsTable({
    tournaments,
}: {
    tournaments: Tournament[];
}) {
    const columns = [
        {
            key: "event",
            label: "Event",
            render: (t: Tournament) => (
                <Link className="table-link" href={`/tournaments/${t.id}`}>
                    <div className="table-title">{t.name}</div>
                    <div className="muted tiny">{t.meta_name}</div>
                </Link>
            ),
            sortValue: (t: Tournament) => t.name,
        },
        {
            key: "date",
            label: "Date",
            render: (t: Tournament) => formatDate(t.date),
        },
        {
            key: "players",
            label: "Players",
            render: (t: Tournament) => t.players.toLocaleString(),
        },
        {
            key: "source",
            label: "Source",
            render: (t: Tournament) => (
                <span className={`badge ${t.is_online ? "badge--online" : ""}`}>
                    {t.is_online ? "Online" : "In person"}
                </span>
            ),
            sortValue: (t: Tournament) => (t.is_online ? 1 : 0),
        },
        {
            key: "winner_archetype",
            label: "Winner archetype",
            render: (t: Tournament) =>
                t.winner_archetype ?? (
                    <span className="muted tiny">Unknown</span>
                ),
        },
    ];

    return <Table columns={columns} rows={tournaments} />;
}
