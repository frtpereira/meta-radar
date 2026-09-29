import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { DecklistView } from "./DecklistView";
import { DECKLIST_VIEW_STORAGE_KEY } from "@/lib/decklist-view-preference";
import type { Card } from "@/lib/types";

const CARDS: Card[] = [
    { name: "Carvanha", set: "PFL", number: "60", count: 4, category: "pokemon" },
    { name: "Ultra Ball", set: "MEG", number: "131", count: 4, category: "trainer" },
];

function renderView() {
    return render(
        React.createElement(DecklistView, {
            cards: CARDS,
            images: {},
            filename: "deck.txt",
        }),
    );
}

describe("DecklistView", () => {
    it("shows the image view by default, with no list tables", () => {
        renderView();

        expect(screen.queryByRole("table")).not.toBeInTheDocument();
        expect(
            screen.getByRole("button", { name: "View as List" }),
        ).toBeInTheDocument();
    });

    it("switches to the list on toggle, and back again", () => {
        renderView();

        fireEvent.click(screen.getByRole("button", { name: "View as List" }));

        // One <table> per category (Pokémon, Trainer) in list view.
        expect(screen.getAllByRole("table")).toHaveLength(2);
        expect(
            screen.getByRole("button", { name: "View as Image" }),
        ).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "View as Image" }));

        expect(screen.queryByRole("table")).not.toBeInTheDocument();
    });

    it("stores the new mode every time the user toggles", () => {
        renderView();

        fireEvent.click(screen.getByRole("button", { name: "View as List" }));
        expect(window.localStorage.getItem(DECKLIST_VIEW_STORAGE_KEY)).toBe(
            "list",
        );

        fireEvent.click(screen.getByRole("button", { name: "View as Image" }));
        expect(window.localStorage.getItem(DECKLIST_VIEW_STORAGE_KEY)).toBe(
            "image",
        );
    });

    it("does not write anything to storage until the user toggles", () => {
        renderView();

        expect(
            window.localStorage.getItem(DECKLIST_VIEW_STORAGE_KEY),
        ).toBeNull();
    });

    it("opens in list view when the last stored preference was list", () => {
        window.localStorage.setItem(DECKLIST_VIEW_STORAGE_KEY, "list");

        renderView();

        expect(screen.getAllByRole("table")).toHaveLength(2);
        expect(
            screen.getByRole("button", { name: "View as Image" }),
        ).toBeInTheDocument();
    });

    it("opens in image view when the last stored preference was image", () => {
        window.localStorage.setItem(DECKLIST_VIEW_STORAGE_KEY, "image");

        renderView();

        expect(screen.queryByRole("table")).not.toBeInTheDocument();
    });

    it("falls back to the default when the stored value is invalid", () => {
        window.localStorage.setItem(DECKLIST_VIEW_STORAGE_KEY, "grid");

        renderView();

        expect(screen.queryByRole("table")).not.toBeInTheDocument();
        expect(
            screen.getByRole("button", { name: "View as List" }),
        ).toBeInTheDocument();
    });

    it("still toggles when localStorage throws", () => {
        vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
            throw new Error("blocked");
        });
        vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
            throw new Error("blocked");
        });

        renderView();
        fireEvent.click(screen.getByRole("button", { name: "View as List" }));

        expect(screen.getAllByRole("table")).toHaveLength(2);
    });
});
