"use client";

import Link from "next/link";
import Table from "@/components/table";
import ArchetypeIcons from "@/components/archetype-icons";
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

    return Math.round(value).toLocaleString("en-US");
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
                    key: "players",
                    label: "Players",
                    render: (t: Tournament) =>
                        t.players.toLocaleString("en-US"),
                },
                {
                    key: "winner_archetype",
                    label: "Winner",
                    render: (t: Tournament) =>
                        t.winner_archetype &&
                        t.winner_nickname &&
                        t.winner_decklist_id ? (
                            <Link href={`/decklists/${t.winner_decklist_id}`}>
                                <ArchetypeIcons
                                    icons={t.winner_archetype_icons}
                                    name={t.winner_archetype}
                                />
                            </Link>
                        ) : (
                            <span className="muted tiny">Unknown</span>
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
                    render: (stat: ArchetypeStat) => (
                        <Link
                            className="table-link"
                            href={`/archetypes/${stat.id}${activeMetaId ? `?meta_id=${activeMetaId}` : ""}`}
                        >
                            <ArchetypeIcons
                                icons={stat.archetype_icons}
                                name={stat.name}
                            />
                        </Link>
                    ),
                },
                {
                    key: "matches",
                    label: "Matches",
                    render: (s: ArchetypeStat) =>
                        s.matches.toLocaleString("en-US"),
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
