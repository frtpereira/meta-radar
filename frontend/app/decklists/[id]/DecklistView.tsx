"use client";

import { useState } from "react";
import Card from "@/components/card";
import { DecklistCategory } from "./DecklistCardsTable";
import { DecklistImageGrid } from "./DecklistImageGrid";
import { ExportDecklistButton } from "./ExportDecklistButton";
import type { Card as CardType } from "@/lib/types";

const CATEGORIES: { key: string; label: string }[] = [
    { key: "pokemon", label: "Pokémon" },
    { key: "trainer", label: "Trainer" },
    { key: "energy", label: "Energy" },
];

// Owns the list/image toggle so both the header button and the body it
// switches live in the same Client Component -- the page itself stays
// a Server Component and only hands down the already-resolved cards
// and image map (see getCardImages in the page).
export function DecklistView({
    cards,
    images,
    filename,
}: {
    cards: CardType[];
    images: Record<string, string>;
    filename: string;
}) {
    const [view, setView] = useState<"image" | "list">("image");

    return (
        <Card
            className="section--spaced"
            heading={
                <>
                    <p className="eyebrow">Decklist</p>
                    <h2>Exact List</h2>
                </>
            }
            headingMeta={
                <span className="decklist-view-actions">
                    <button
                        type="button"
                        className="button button--active"
                        aria-pressed={view === "list"}
                        onClick={() =>
                            setView((v) => (v === "image" ? "list" : "image"))
                        }
                    >
                        {view === "image" ? "View as List" : "View as Image"}
                    </button>
                    <ExportDecklistButton cards={cards} filename={filename} />
                </span>
            }
        >
            {view === "list" ? (
                <div className="grid grid--three">
                    {CATEGORIES.map(({ key, label }) => (
                        <DecklistCategory
                            key={key}
                            label={label}
                            cards={cards.filter((c) => c.category === key)}
                            images={images}
                        />
                    ))}
                </div>
            ) : (
                <DecklistImageGrid cards={cards} images={images} />
            )}
        </Card>
    );
}
