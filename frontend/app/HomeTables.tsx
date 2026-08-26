"use client";

import Link from "next/link";
import Table from "@/components/table";
import type { ArchetypeStat, Tournament } from "@/lib/types";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so the column definitions for the
// home page's tables live here, in a client module, rather than inline in
// the (Server Component) page.
function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
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

export function LiveTournamentsTable({
    tournaments,
}: {
    tournaments: Tournament[];
}) {
    return (
        <Table
            columns={[
                {
                    key: "event",
                    label: "Event",
                    render: (t: Tournament) => (
                        <Link
                            className="table-link"
                            href={`/tournaments/${t.id}`}
                        >
                            <div className="table-title">{t.name}</div>
                        </Link>
                    ),
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
                        <span
                            className={`badge ${t.is_online ? "badge--online" : ""}`}
                        >
                            {t.is_online ? "Online" : "In person"}
                        </span>
                    ),
                },
            ]}
            rows={tournaments}
            sortable={false}
        />
    );
}

export function TopArchetypesTable({
    archetypes,
    activeMetaId,
}: {
    archetypes: ArchetypeStat[];
    activeMetaId: string | null;
}) {
    return (
        <Table
            columns={[
                {
                    key: "archetype",
                    label: "Archetype",
                    render: (stat: ArchetypeStat) =>
                        activeMetaId ? (
                            <Link
                                className="table-link"
                                href={`/matchups?meta_id=${activeMetaId}&archetype_id=${stat.id}`}
                            >
                                <div className="table-title">{stat.name}</div>
                            </Link>
                        ) : (
                            <div className="table-title">{stat.name}</div>
                        ),
                },
                {
                    key: "decks",
                    label: "Decks",
                    render: (s: ArchetypeStat) =>
                        s.deck_count.toLocaleString(),
                },
                {
                    key: "avg",
                    label: "Avg standing",
                    render: (s: ArchetypeStat) =>
                        formatStanding(s.avg_standing),
                },
                {
                    key: "win",
                    label: "Win rate",
                    render: (s: ArchetypeStat) => formatPercent(s.win_rate),
                },
            ]}
            rows={archetypes}
            sortable={false}
        />
    );
}
