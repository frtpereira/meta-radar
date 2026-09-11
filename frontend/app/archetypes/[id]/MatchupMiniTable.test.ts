import { render, screen, within } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import type { MatchupStat } from "@/lib/types";
import { MatchupMiniTable } from "./DecklistTables";

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
            icons: null,
        },
        opponent: {
            id: opponentId,
            name: `Opponent ${opponentId}`,
            slug: `opponent-${opponentId}`,
            icons: null,
        },
        matches,
        wins,
        losses,
        ties,
        win_rate: winRate,
        score_rate: scoreRate,
    };
}

describe("MatchupMiniTable", () => {
    it("renders deck icons for the opponent column", () => {
        const stat = makeMatchup(1, 2, 24, 14, 8, 2, 0.636, 0.583);
        stat.opponent.icons = ["charizard"];

        render(
            React.createElement(MatchupMiniTable, {
                stats: [stat],
                archetypeId: 1,
                label: "Best matchups",
                variant: "good",
                metaId: "meta-1",
            }),
        );

        const row = within(screen.getByRole("table")).getAllByRole("row")[1];
        const cells = within(row).getAllByRole("cell");

        const opponentIcon = within(cells[0]).getByRole("img");
        expect(opponentIcon).toHaveAttribute(
            "src",
            expect.stringContaining("/charizard.png"),
        );
        expect(within(cells[0]).getByText("Opponent 2")).toBeInTheDocument();
    });

    it("falls back to the unknown icon when the opponent has no curated icons", () => {
        const stat = makeMatchup(1, 2, 24, 14, 8, 2, 0.636, 0.583);
        stat.opponent.icons = null;

        render(
            React.createElement(MatchupMiniTable, {
                stats: [stat],
                archetypeId: 1,
                label: "Best matchups",
                variant: "good",
                metaId: "meta-1",
            }),
        );

        const row = within(screen.getByRole("table")).getAllByRole("row")[1];
        const cells = within(row).getAllByRole("cell");

        const opponentIcon = within(cells[0]).getByRole("img");
        expect(opponentIcon).toHaveAttribute(
            "src",
            expect.stringContaining("/unknown.png"),
        );
    });
});
