import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { ExportDecklistButton } from "./ExportDecklistButton";
import type { Card } from "@/lib/types";

const CARDS: Card[] = [
    { name: "Carvanha", set: "PFL", number: "60", count: 4, category: "pokemon" },
    { name: "Ultra Ball", set: "MEG", number: "131", count: 4, category: "trainer" },
    { name: "Darkness Energy", set: "MEE", number: "7", count: 9, category: "energy" },
];

function mockClipboard(writeText: (text: string) => Promise<void>) {
    Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: { writeText },
    });
}

describe("ExportDecklistButton", () => {
    it("copies the formatted decklist to the clipboard on click", async () => {
        const writeText = vi.fn().mockResolvedValue(undefined);
        mockClipboard(writeText);

        render(React.createElement(ExportDecklistButton, { cards: CARDS }));

        fireEvent.click(screen.getByRole("button", { name: "Export Decklist" }));

        await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
        const copied = writeText.mock.calls[0][0] as string;
        expect(copied).toContain("Pokémon: 4");
        expect(copied).toContain("4 Carvanha PFL 60");
        expect(copied).toContain("Trainer: 4");
        expect(copied).toContain("Energy: 9");
    });

    it("shows 'Copied!' feedback then reverts after the delay", async () => {
        vi.useFakeTimers();
        try {
            mockClipboard(() => Promise.resolve());

            render(React.createElement(ExportDecklistButton, { cards: CARDS }));

            await act(async () => {
                fireEvent.click(
                    screen.getByRole("button", { name: "Export Decklist" }),
                );
                // Let the pending clipboard promise resolve before the
                // component schedules its reset timer.
                await Promise.resolve();
                await Promise.resolve();
            });

            expect(
                screen.getByRole("button", { name: "Copied!" }),
            ).toBeInTheDocument();

            await act(async () => {
                vi.advanceTimersByTime(2000);
            });

            expect(
                screen.getByRole("button", { name: "Export Decklist" }),
            ).toBeInTheDocument();
        } finally {
            vi.useRealTimers();
        }
    });

    it("falls back to a file download when the Clipboard API is unavailable", async () => {
        Object.defineProperty(navigator, "clipboard", {
            configurable: true,
            value: undefined,
        });

        const createObjectURL = vi.fn().mockReturnValue("blob:mock-url");
        const revokeObjectURL = vi.fn();
        Object.defineProperty(URL, "createObjectURL", {
            configurable: true,
            value: createObjectURL,
        });
        Object.defineProperty(URL, "revokeObjectURL", {
            configurable: true,
            value: revokeObjectURL,
        });
        const clickSpy = vi
            .spyOn(HTMLAnchorElement.prototype, "click")
            .mockImplementation(() => {});

        render(
            React.createElement(ExportDecklistButton, {
                cards: CARDS,
                filename: "my-deck.txt",
            }),
        );

        fireEvent.click(screen.getByRole("button", { name: "Export Decklist" }));

        await waitFor(() => expect(clickSpy).toHaveBeenCalledTimes(1));
        expect(createObjectURL).toHaveBeenCalledTimes(1);
        expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    });
});
