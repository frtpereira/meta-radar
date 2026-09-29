"use client";

import { useEffect, useState } from "react";
import Card from "@/components/card";
import { DecklistCategory } from "./DecklistCardsTable";
import { DecklistImageGrid } from "./DecklistImageGrid";
import { ExportDecklistButton } from "./ExportDecklistButton";
import type { Card as CardType } from "@/lib/types";
import {
    DEFAULT_DECKLIST_VIEW,
    readDecklistViewMode,
    writeDecklistViewMode,
    type DecklistViewMode,
} from "@/lib/decklist-view-preference";

const CATEGORIES: { key: string; label: string }[] = [
    { key: "pokemon", label: "Pokémon" },
    { key: "trainer", label: "Trainer" },
    { key: "energy", label: "Energy" },
];

// Owns the list/image toggle so both the header button and the body it
// switches live in the same Client Component -- the page itself stays
// a Server Component and only hands down the already-resolved cards
// and image map (see getCardImages in the page). The last choice is
// remembered in localStorage, so every decklist opens the way the user
// left it.
export function DecklistView({
    cards,
    images,
    filename,
}: {
    cards: CardType[];
    images: Record<string, string>;
    filename: string;
}) {
    // Server render and first client render both use the default so
    // hydration matches; the stored preference (which only exists in the
    // browser) is applied right after mount.
    const [view, setView] = useState<DecklistViewMode>(DEFAULT_DECKLIST_VIEW);

    useEffect(() => {
        setView(readDecklistViewMode());
    }, []);

    function toggleView() {
        const next: DecklistViewMode = view === "image" ? "list" : "image";
        setView(next);
        writeDecklistViewMode(next);
    }

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
                        onClick={toggleView}
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
