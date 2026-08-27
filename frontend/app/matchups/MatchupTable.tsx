"use client";

import type { MatchupStat } from "@/lib/types";
import Table from "@/components/table";
import InfoTooltip from "@/components/info-tooltip";

// Table `columns` entries carry `render`/`sortValue` functions, and Table
// itself is a Client Component (for sort state). Functions can't cross the
// Server -> Client Component boundary, so this column configuration lives
// here, in a client module, rather than inline in the (Server Component)
// matchups page.
const PAGE_SIZE = 20;

function formatPercent(value: number | null) {
    if (value === null) {
        return "—";
    }

    return `${Math.round(value * 1000) / 10}%`;
}

export default function MatchupTable({
    stats,
    selectedArchetypeId,
}: {
    stats: MatchupStat[];
    selectedArchetypeId: string;
}) {
    const columns = [
        {
            key: "primary",
            label: "Archetype",
            // This column shows the currently-selected archetype's
            // perspective, which flips per row depending on which side of
            // the pairing the selected archetype was on -- not a stable
            // sort key, so leave it unsortable.
            sortable: false,
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                const primary = selectedIsArchetype
                    ? stat.archetype
                    : stat.opponent;
                return <div className="table-title">{primary.name}</div>;
            },
        },
        {
            key: "secondary",
            label: "Opponent",
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                const secondary = selectedIsArchetype
                    ? stat.opponent
                    : stat.archetype;
                return <div className="table-title">{secondary.name}</div>;
            },
            sortValue: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype
                    ? stat.opponent.name
                    : stat.archetype.name;
            },
        },
        {
            key: "record",
            label: "Record",
            // The displayed W-L-T string depends on perspective/mirror
            // status and isn't a single meaningful sort key, so leave this
            // column unsortable.
            sortable: false,
            render: (stat: MatchupStat) => {
                // W-L is meaningless for a mirror: both sides are the same
                // archetype, so wins and losses are equal by definition.
                const isMirror = stat.archetype.id === stat.opponent.id;
                if (isMirror) {
                    return `${stat.ties} ties`;
                }

                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                const displayedWins = selectedIsArchetype
                    ? stat.wins
                    : stat.losses;
                const displayedLosses = selectedIsArchetype
                    ? stat.losses
                    : stat.wins;
                const displayedTies = stat.ties;
                return `${displayedWins}-${displayedLosses}-${displayedTies}`;
            },
        },
        {
            key: "matches",
            label: "Matches",
            sortDescFirst: true,
            render: (stat: MatchupStat) => stat.matches.toLocaleString(),
            sortValue: (stat: MatchupStat) => stat.matches,
        },
        {
            key: "score_rate",
            label: (
                <>
                    Score rate
                    <InfoTooltip text="Share of possible match points earned in this matchup, from the highlighted archetype's perspective: (wins + 0.5 × ties) ÷ matches played. Unlike win rate, ties count as half a win instead of being excluded." />
                </>
            ),
            sortDescFirst: true,
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                // score_rate is stored from `stat.archetype`'s perspective
                // and is always a fraction in [0, 1]. When the row is being
                // displayed from the opponent's perspective, the correct
                // complement is `1 - score_rate` (matches the win_rate swap
                // above), not `-score_rate` — negating it produced bogus
                // negative percentages.
                const displayedScoreRate =
                    stat.score_rate === null
                        ? null
                        : selectedIsArchetype
                          ? stat.score_rate
                          : 1 - stat.score_rate;
                return formatPercent(displayedScoreRate);
            },
            sortValue: (stat: MatchupStat) => {
                if (stat.score_rate === null) {
                    return null;
                }
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype
                    ? stat.score_rate
                    : 1 - stat.score_rate;
            },
        },
        {
            key: "win_rate",
            label: "Win rate",
            sortDescFirst: true,
            render: (stat: MatchupStat) => {
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                // win_rate is null for mirrors (see backend) -- trust that
                // instead of recomputing from wins/losses, which are equal
                // for a mirror and would otherwise render a bogus 50%.
                const displayedWinRate =
                    stat.win_rate === null
                        ? null
                        : selectedIsArchetype
                          ? stat.win_rate
                          : 1 - stat.win_rate;
                return formatPercent(displayedWinRate);
            },
            sortValue: (stat: MatchupStat) => {
                if (stat.win_rate === null) {
                    return null;
                }
                const selectedIsArchetype =
                    String(stat.archetype.id) === String(selectedArchetypeId);
                return selectedIsArchetype ? stat.win_rate : 1 - stat.win_rate;
            },
        },
    ];

    return <Table columns={columns} rows={stats} pageSize={PAGE_SIZE} />;
}
