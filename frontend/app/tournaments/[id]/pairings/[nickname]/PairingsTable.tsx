"use client";

import Link from "next/link";
import Table from "@/components/table";
import ArchetypeIcons from "@/components/archetype-icons";
import type { PairingOutcome, PairingRow } from "@/lib/types";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so this column configuration lives
// here, in a client module, rather than inline in the (Server Component)
// pairings page.
function formatRound(r: PairingRow) {
    // Most tournaments have a single phase, so the common case is just
    // "Round N" -- multi-phase events (e.g. Swiss into Top Cut) get the
    // phase called out too, since round numbers reset per phase.
    return r.phase > 1
        ? `Round ${r.round} (Phase ${r.phase})`
        : `Round ${r.round}`;
}

function outcomeBadgeClass(outcome: PairingOutcome) {
    switch (outcome) {
        case "win":
            return "badge badge--win";
        case "loss":
            return "badge badge--loss";
        case "draw":
            return "badge badge--draw";
        default:
            return "badge";
    }
}

function formatOutcome(outcome: PairingOutcome) {
    switch (outcome) {
        case "win":
            return "Win";
        case "loss":
            return "Loss";
        case "draw":
            return "Draw";
        case "bye":
            return "Bye";
        default:
            return "Unknown";
    }
}

export default function PairingsTable({
    pairings,
}: {
    pairings: PairingRow[];
}) {
    return (
        <Table
            sortable={false}
            columns={[
                {
                    key: "round",
                    label: "Round",
                    sortable: false,
                    render: (r: PairingRow) => formatRound(r),
                },
                {
                    key: "opponent",
                    label: "Opponent",
                    sortable: false,
                    render: (r: PairingRow) =>
                        r.opponent_name != null ? (
                            <Link
                                className="table-link"
                                href={`/players/${encodeURIComponent(r.opponent_name)}`}
                            >
                                <div className="table-title">
                                    {r.opponent_name}
                                </div>
                            </Link>
                        ) : (
                            <span className="muted tiny">—</span>
                        ),
                },
                {
                    key: "archetype",
                    label: "Archetype",
                    sortable: false,
                    render: (r: PairingRow) =>
                        r.opponent_name != null ? (
                            <ArchetypeIcons
                                icons={r.opponent_archetype_icons}
                                name={r.opponent_archetype_name ?? "Unknown"}
                            />
                        ) : (
                            <span className="muted tiny">—</span>
                        ),
                },
                {
                    key: "result",
                    label: "Result",
                    sortable: false,
                    render: (r: PairingRow) => (
                        <span className={outcomeBadgeClass(r.outcome)}>
                            {formatOutcome(r.outcome)}
                        </span>
                    ),
                },
                {
                    key: "decklist",
                    label: "Opponent Decklist",
                    sortable: false,
                    render: (r: PairingRow) =>
                        r.opponent_decklist_id != null ? (
                            <Link
                                className="button"
                                href={`/decklists/${r.opponent_decklist_id}`}
                            >
                                Decklist
                            </Link>
                        ) : (
                            <span className="muted tiny">—</span>
                        ),
                },
            ]}
            rows={pairings}
        />
    );
}
