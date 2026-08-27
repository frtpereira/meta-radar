"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import Table, { type SortState } from "@/components/table";
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
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();

    // Sorting is server-side (see ListTournaments): `tournaments` is only
    // ever the current page, so reordering it locally could never sort
    // the full result set -- clicking a header re-fetches with the new
    // sort_by/sort_dir instead of reordering what's already on screen.
    const sortByParam = searchParams?.get("sort_by") ?? "";
    const sortDirParam =
        searchParams?.get("sort_dir") === "asc" ? "asc" : "desc";
    const sortState: SortState = sortByParam
        ? { key: sortByParam, direction: sortDirParam }
        : null;

    function handleSortChange(column: { key: string }) {
        const sp = new URLSearchParams(searchParams?.toString() ?? "");

        // Date and players are both more useful highest-first, so the
        // first click sorts descending instead of the usual ascending.
        if (sortState && sortState.key === column.key) {
            if (sortState.direction === "desc") {
                sp.set("sort_by", column.key);
                sp.set("sort_dir", "asc");
            } else {
                sp.delete("sort_by");
                sp.delete("sort_dir");
            }
        } else {
            sp.set("sort_by", column.key);
            sp.set("sort_dir", "desc");
        }
        // the current page position belongs to the old order, not the new one
        sp.delete("page");

        router.replace(`${pathname}?${sp.toString()}`);
    }

    const columns = [
        {
            key: "event",
            label: "Event",
            // Name isn't in the server-side sort whitelist.
            sortable: false,
            render: (t: Tournament) => (
                <Link className="table-link" href={`/tournaments/${t.id}`}>
                    <div className="table-title">{t.name}</div>
                    <div className="muted tiny">{t.meta_name}</div>
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
            // Online/in-person isn't in the server-side sort whitelist.
            sortable: false,
            render: (t: Tournament) => (
                <span className={`badge ${t.is_online ? "badge--online" : ""}`}>
                    {t.is_online ? "Online" : "In person"}
                </span>
            ),
        },
        {
            key: "winner_archetype",
            label: "Winner archetype",
            // A winner's archetype changes meaning across rows (some are
            // decisive, some default to whoever placed first with no
            // clean tiebreak) and isn't a useful global sort key.
            sortable: false,
            render: (t: Tournament) =>
                t.winner_archetype ?? (
                    <span className="muted tiny">Unknown</span>
                ),
        },
    ];

    return (
        <Table
            columns={columns}
            rows={tournaments}
            sortState={sortState}
            onSortChange={handleSortChange}
        />
    );
}
