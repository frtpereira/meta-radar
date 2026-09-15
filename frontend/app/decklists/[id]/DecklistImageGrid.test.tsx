import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import { DecklistImageGrid } from "./DecklistImageGrid";
import type { Card } from "@/lib/types";

// Deliberately out of category order (energy, then pokemon, then
// trainer) and, within pokemon, in the order a player might have
// entered them -- the grid must re-group by category without
// reordering cards inside a category.
const CARDS: Card[] = [
    { name: "Darkness Energy", set: "MEE", number: "7", count: 9, category: "energy" },
    { name: "Carvanha", set: "PFL", number: "60", count: 4, category: "pokemon" },
    { name: "Sharpedo ex", set: "PFL", number: "61", count: 2, category: "pokemon" },
    { name: "Ultra Ball", set: "MEG", number: "131", count: 4, category: "trainer" },
];

const IMAGES: Record<string, string> = {
    "PFL:60": "https://img.example/carvanha.png",
    "PFL:61": "https://img.example/sharpedo.png",
    "MEG:131": "https://img.example/ultra-ball.png",
    // "MEE:7" deliberately missing to exercise the placeholder path.
};

describe("DecklistImageGrid", () => {
    it("groups cards pokemon -> trainer -> energy, preserving order within a category", () => {
        render(
            React.createElement(DecklistImageGrid, {
                cards: CARDS,
                images: IMAGES,
            }),
        );

        const names = screen
            .getAllByRole("img", { hidden: true })
            .map((img) => img.getAttribute("alt"));

        expect(names).toEqual(["Carvanha", "Sharpedo ex", "Ultra Ball"]);
    });

    it("shows each card's count as a badge, including counts above 4", () => {
        render(
            React.createElement(DecklistImageGrid, {
                cards: CARDS,
                images: IMAGES,
            }),
        );

        // Carvanha and Ultra Ball are both 4-of, so two badges read "4".
        expect(screen.getAllByText("4", { selector: "svg text" })).toHaveLength(2);
        expect(screen.getByText("2", { selector: "svg text" })).toBeInTheDocument();
        expect(screen.getByText("9", { selector: "svg text" })).toBeInTheDocument();
    });

    it("falls back to a text placeholder when no image was resolved", () => {
        render(
            React.createElement(DecklistImageGrid, {
                cards: CARDS,
                images: IMAGES,
            }),
        );

        expect(screen.getByText("Darkness Energy")).toBeInTheDocument();
    });
});
