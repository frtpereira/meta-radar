import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import type { ArchetypeStat } from "@/lib/types";
import ArchetypeSearch from "./ArchetypeSearch";

vi.mock("next/link", () => ({
    default: (props: any) =>
        React.createElement(
            "a",
            { href: props.href, ...props },
            props.children,
        ),
}));

function makeArchetype(id: number, name: string): ArchetypeStat {
    return {
        id,
        name,
        slug: name.toLowerCase().replace(/\s+/g, "-"),
        deck_count: 20 + id,
        avg_standing: id,
        drop_count: 0,
        matches: 10 + id,
        wins: 5 + id,
        losses: 2,
        ties: 1,
        score_rate: 0.5,
        win_rate: 0.5,
    };
}

const archetypes = [
    ...Array.from({ length: 24 }, (_, index) =>
        makeArchetype(
            index + 1,
            `Archetype ${String(index + 1).padStart(2, "0")}`,
        ),
    ),
    makeArchetype(25, "Late Game Dragon"),
];

describe("ArchetypeSearch", () => {
    it("renders the first page of results and paginates locally", async () => {
        const user = userEvent.setup();
        render(
            React.createElement(ArchetypeSearch, {
                archetypes,
                metaId: "meta-1",
            }),
        );

        expect(screen.getByText("25 archetypes")).toBeInTheDocument();
        expect(screen.getByText("Page 1 of 2")).toBeInTheDocument();
        expect(screen.getByText("Archetype 01")).toBeInTheDocument();
        expect(screen.queryByText("Late Game Dragon")).not.toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: "Next" }));

        expect(screen.getByText("Page 2 of 2")).toBeInTheDocument();
        expect(screen.getByText("Late Game Dragon")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Prev" })).toBeEnabled();
    });

    it("filters case-insensitively across the full result set and resets pagination", async () => {
        const user = userEvent.setup();
        render(
            React.createElement(ArchetypeSearch, {
                archetypes,
                metaId: "meta-1",
            }),
        );

        await user.click(screen.getByRole("button", { name: "Next" }));
        await user.type(
            screen.getByRole("searchbox", { name: "Search archetypes" }),
            "  late game ",
        );

        expect(screen.getByText("1 archetypes")).toBeInTheDocument();
        expect(screen.getByText("Late Game Dragon")).toBeInTheDocument();
        expect(screen.queryByText(/Page \d of \d/)).not.toBeInTheDocument();
        expect(
            screen.queryByRole("button", { name: "Next" }),
        ).not.toBeInTheDocument();
    });

    it("links each archetype row to the matching detail page with the active meta id", () => {
        render(
            React.createElement(ArchetypeSearch, {
                archetypes,
                metaId: "meta-1",
            }),
        );

        expect(
            screen.getByRole("link", { name: "Archetype 01" }),
        ).toHaveAttribute("href", "/archetypes/1?meta_id=meta-1");
    });

    it("renders an empty state when no archetypes match the query", async () => {
        const user = userEvent.setup();
        render(
            React.createElement(ArchetypeSearch, {
                archetypes,
                metaId: "meta-1",
            }),
        );

        await user.type(
            screen.getByRole("searchbox", { name: "Search archetypes" }),
            "unknown deck",
        );

        expect(screen.getByText("No matching archetypes")).toBeInTheDocument();
        expect(
            screen.getByText('No archetypes match "unknown deck".'),
        ).toBeInTheDocument();
    });
});
