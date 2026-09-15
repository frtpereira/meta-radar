"use client";

import { CountBadge } from "@/components/count-badge";
import type { Card } from "@/lib/types";

// Same category ordering as the list view (DecklistCardsTable) and the
// plain-text export (lib/decklist-export) -- kept in one place there
// would be nice, but it's three call sites each already tied to their
// own rendering, so duplicating the tuple is simpler than threading a
// shared constant through server/client boundaries for three lines.
const CATEGORY_ORDER = ["pokemon", "trainer", "energy"] as const;

// One tile per unique card, count shown as a badge rather than
// repeating the image `count` times -- keeps a 60-card deck to a
// glance-able grid instead of a wall of duplicate art.
function CardImageTile({ card, imageUrl }: { card: Card; imageUrl?: string }) {
    return (
        <div className="decklist-image-tile">
            {imageUrl ? (
                <img src={imageUrl} alt={card.name} loading="lazy" />
            ) : (
                <div className="decklist-image-tile__placeholder">
                    {card.name}
                </div>
            )}
            <CountBadge
                count={card.count}
                className="decklist-image-tile__badge"
            />
        </div>
    );
}

export function DecklistImageGrid({
    cards,
    images,
}: {
    cards: Card[];
    images: Record<string, string>;
}) {
    // Cards keep the order they were dealt in within each category --
    // only grouped by category, never re-sorted -- per issue #107.
    const ordered = CATEGORY_ORDER.flatMap((category) =>
        cards.filter((c) => c.category === category),
    );

    return (
        <div className="decklist-image-grid">
            {ordered.map((card, i) => (
                <CardImageTile
                    key={`${card.set}:${card.number}:${i}`}
                    card={card}
                    imageUrl={images[`${card.set}:${card.number}`]}
                />
            ))}
        </div>
    );
}
