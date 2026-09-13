import { describe, expect, it } from "vitest";
import { formatDecklistExport } from "./decklist-export";
import type { Card } from "./types";

const CARDS: Card[] = [
    { name: "Carvanha", set: "PFL", number: "60", count: 4, category: "pokemon" },
    { name: "Yveltal", set: "MEG", number: "88", count: 1, category: "pokemon" },
    {
        name: "Lillie's Determination",
        set: "MEG",
        number: "119",
        count: 4,
        category: "trainer",
    },
    { name: "Ultra Ball", set: "MEG", number: "131", count: 4, category: "trainer" },
    { name: "Darkness Energy", set: "MEE", number: "7", count: 9, category: "energy" },
];

describe("formatDecklistExport", () => {
    it("groups cards by category in Pokémon / Trainer / Energy order", () => {
        const result = formatDecklistExport(CARDS);
        const sections = result.split("\n\n");

        expect(sections).toHaveLength(3);
        expect(sections[0].startsWith("Pokémon:")).toBe(true);
        expect(sections[1].startsWith("Trainer:")).toBe(true);
        expect(sections[2].startsWith("Energy:")).toBe(true);
    });

    it("headers each section with the summed card count", () => {
        const result = formatDecklistExport(CARDS);

        expect(result).toContain("Pokémon: 5");
        expect(result).toContain("Trainer: 8");
        expect(result).toContain("Energy: 9");
    });

    it("formats each line as '<count> <name> <set> <number>'", () => {
        const result = formatDecklistExport(CARDS);

        expect(result).toContain("4 Carvanha PFL 60");
        expect(result).toContain("1 Yveltal MEG 88");
        expect(result).toContain("9 Darkness Energy MEE 7");
    });

    it("omits empty categories entirely", () => {
        const pokemonOnly: Card[] = [CARDS[0]];
        const result = formatDecklistExport(pokemonOnly);

        expect(result).not.toContain("Trainer:");
        expect(result).not.toContain("Energy:");
        expect(result).toBe("Pokémon: 4\n4 Carvanha PFL 60");
    });

    it("omits the set/number suffix when a card has no set", () => {
        const noSet: Card[] = [
            { name: "Mystery Card", set: "", number: "", count: 1, category: "pokemon" },
        ];

        expect(formatDecklistExport(noSet)).toBe("Pokémon: 1\n1 Mystery Card");
    });

    it("returns an empty string for an empty decklist", () => {
        expect(formatDecklistExport([])).toBe("");
    });
});
