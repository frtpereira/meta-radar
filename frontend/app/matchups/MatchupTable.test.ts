import { render, screen, within } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import type { MatchupStat } from "@/lib/types";
import MatchupTable from "./MatchupTable";

function makeMatchup(
    archetypeId: number,
    opponentId: number,
    matches: number,
    wins: number,
    losses: number,
    ties: number,
    winRate: number,
    scoreRate: number,
): MatchupStat {
    return {
        archetype: {
            id: archetypeId,
            name: `Archetype ${archetypeId}`,
            slug: `archetype-${archetypeId}`,
        },
        opponent: {
            id: opponentId,
            name: `Opponent ${opponentId}`,
            slug: `opponent-${opponentId}`,
        },
        matches,
        wins,
        losses,
        ties,
        win_rate: winRate,
        score_rate: scoreRate,
    };
}

describe("MatchupTable", () => {
    it("renders the normalized column order", () => {
        render(
            React.createElement(MatchupTable, {
                selectedArchetypeId: "1",
                stats: [
                    makeMatchup(1, 2, 24, 14, 8, 2, 0.636, 0.583),
                ],
            }),
        );

        const headers = within(screen.getByRole("table")).getAllByRole(
            "columnheader",
        );

        expect(headers[0]).toHaveTextContent("Archetype");
        expect(headers[1]).toHaveTextContent("Opponent");
        expect(headers[2]).toHaveTextContent("Matches");
        expect(headers[3]).toHaveTextContent("Win rate");
        expect(headers[4]).toHaveTextContent("Score rate");
        expect(headers[5]).toHaveTextContent("Record");
    });

    it("applies the decklists win-rate coloring", () => {
        render(
            React.createElement(MatchupTable, {
                selectedArchetypeId: "1",
                stats: [
                    makeMatchup(1, 2, 10, 6, 4, 0, 0.6, 0.6),
                    makeMatchup(1, 3, 10, 4, 6, 0, 0.4, 0.4),
                ],
            }),
        );

        const rows = within(screen.getByRole("table")).getAllByRole("row").slice(1);

        expect(
            within(rows[0]).getAllByRole("cell")[3].querySelector("span"),
        ).toHaveStyle({ color: "var(--success)", fontWeight: "600" });
        expect(
            within(rows[1]).getAllByRole("cell")[3].querySelector("span"),
        ).toHaveStyle({ color: "var(--accent-2)", fontWeight: "600" });
    });
});
