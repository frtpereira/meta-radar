"use client";

import Table from "@/components/table";
import type { TournamentStanding } from "@/lib/types";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so this column configuration lives
// here, in a client module, rather than inline in the (Server Component)
// tournament detail page.
function formatStanding(value: number) {
    return value === 0 ? "Dropped" : `#${value}`;
}

export default function StandingsTable({
    standings,
}: {
    standings: TournamentStanding[];
}) {
    return (
        <Table
            columns={[
                {
                    key: "standing",
                    label: "Standing",
                    render: (r: TournamentStanding) =>
                        formatStanding(r.standing),
                },
                {
                    key: "player",
                    label: "Player",
                    render: (r: TournamentStanding) => (
                        <div className="table-title">{r.player_name}</div>
                    ),
                    sortValue: (r: TournamentStanding) => r.player_name,
                },
                {
                    key: "archetype",
                    label: "Archetype",
                    render: (r: TournamentStanding) =>
                        r.archetype_name ?? (
                            <span className="muted tiny">Unknown</span>
                        ),
                    sortValue: (r: TournamentStanding) => r.archetype_name,
                },
                {
                    key: "score",
                    label: "Score",
                    render: (r: TournamentStanding) =>
                        `${r.wins}-${r.losses}-${r.ties}`,
                    sortValue: (r: TournamentStanding) => r.wins - r.losses,
                },
            ]}
            rows={standings}
        />
    );
}
