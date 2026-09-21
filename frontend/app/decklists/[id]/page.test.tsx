import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DecklistDetail } from "@/lib/types";

const getDecklist = vi.fn();
const getCardImages = vi.fn();

vi.mock("@/lib/api", () => ({
    getDecklist: (...args: unknown[]) => getDecklist(...args),
    getCardImages: (...args: unknown[]) => getCardImages(...args),
}));

vi.mock("next/navigation", () => ({
    notFound: () => {
        throw new Error("NEXT_NOT_FOUND");
    },
}));

import PlayerDecklistPage from "./page";

function makeDecklist(overrides: Partial<DecklistDetail> = {}): DecklistDetail {
    return {
        id: 101,
        tournament_id: "tour-01",
        tournament_name: "Worlds Warmup Regional",
        date: "2026-05-01",
        player_id: "p-ash",
        player_name: "Ash Ketchum",
        archetype_id: 1,
        archetype_name: "Charizard ex",
        archetype_slug: "charizard-ex",
        archetype_icons: ["charizard"],
        cards: [
            {
                name: "Charmander",
                set: "PAF",
                number: "7",
                count: 4,
                category: "pokemon",
            },
        ],
        ...overrides,
    };
}

async function renderPage(id = "101") {
    // The route is /decklists/[id]: `id` is the only param Next passes in.
    const ui = await PlayerDecklistPage({ params: Promise.resolve({ id }) });
    return render(ui);
}

describe("decklist page player links", () => {
    beforeEach(() => {
        getCardImages.mockResolvedValue({});
    });

    it("points the back button at the player from the decklist, not the URL", async () => {
        getDecklist.mockResolvedValue(makeDecklist());

        await renderPage();

        const back = screen.getByRole("link", { name: /Back to Ash Ketchum/ });
        expect(back).toHaveAttribute("href", "/players/Ash%20Ketchum");
        expect(back.getAttribute("href")).not.toContain("undefined");
    });

    it("points the player pill at the same profile", async () => {
        getDecklist.mockResolvedValue(makeDecklist());

        await renderPage();

        expect(
            screen.getByRole("link", { name: "Ash Ketchum" }),
        ).toHaveAttribute("href", "/players/Ash%20Ketchum");
    });

    it("URL-encodes nicknames containing reserved characters", async () => {
        getDecklist.mockResolvedValue(
            makeDecklist({ player_name: "A/B #1?" }),
        );

        await renderPage();

        expect(
            screen.getByRole("link", { name: /Back to A\/B #1\?/ }),
        ).toHaveAttribute("href", "/players/A%2FB%20%231%3F");
        expect(screen.getByRole("link", { name: "A/B #1?" })).toHaveAttribute(
            "href",
            "/players/A%2FB%20%231%3F",
        );
    });
});
