import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import { DecklistView } from "./DecklistView";
import type { Card } from "@/lib/types";

const CARDS: Card[] = [
    { name: "Carvanha", set: "PFL", number: "60", count: 4, category: "pokemon" },
    { name: "Ultra Ball", set: "MEG", number: "131", count: 4, category: "trainer" },
];

describe("DecklistView", () => {
    it("shows the list view by default, with an image-view grid absent", () => {
        render(
            React.createElement(DecklistView, {
                cards: CARDS,
                images: {},
                filename: "deck.txt",
            }),
        );

        // One <table> per category (Pokémon, Trainer) in list view.
        expect(screen.getAllByRole("table")).toHaveLength(2);
        expect(
            screen.getByRole("button", { name: "View as Images" }),
        ).toBeInTheDocument();
    });

    it("switches to the image grid on toggle, and back again", () => {
        render(
            React.createElement(DecklistView, {
                cards: CARDS,
                images: {},
                filename: "deck.txt",
            }),
        );

        fireEvent.click(screen.getByRole("button", { name: "View as Images" }));

        expect(screen.queryByRole("table")).not.toBeInTheDocument();
        expect(
            screen.getByRole("button", { name: "View as List" }),
        ).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "View as List" }));

        expect(screen.getAllByRole("table")).toHaveLength(2);
    });
});
