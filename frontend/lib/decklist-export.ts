import type { Card } from "./types";

const CATEGORY_ORDER = ["pokemon", "trainer", "energy"] as const;

const CATEGORY_LABEL: Record<(typeof CATEGORY_ORDER)[number], string> = {
    pokemon: "Pokémon",
    trainer: "Trainer",
    energy: "Energy",
};

function formatCardLine(card: Card): string {
    const setInfo = card.set
        ? ` ${card.set}${card.number ? ` ${card.number}` : ""}`
        : "";
    return `${card.count} ${card.name}${setInfo}`;
}

// Renders a decklist in the plain-text "ptcgo-style" format used by
// tournament tools (e.g. LimitlessTCG), grouped and ordered
// Pokémon -> Trainer -> Energy, each section headed by its card count.
export function formatDecklistExport(cards: Card[]): string {
    const sections = CATEGORY_ORDER.map((category) => {
        const categoryCards = cards.filter((c) => c.category === category);
        if (categoryCards.length === 0) return null;

        const total = categoryCards.reduce((sum, c) => sum + c.count, 0);
        const lines = categoryCards.map(formatCardLine);

        return `${CATEGORY_LABEL[category]}: ${total}\n${lines.join("\n")}`;
    }).filter((section): section is string => section !== null);

    return sections.join("\n\n");
}
