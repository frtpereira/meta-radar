"use client";

import Table from "@/components/table";
import CardHoverPreview from "@/components/card-hover-preview";
import type { Card } from "@/lib/types";

// Table `columns` entries carry `render` functions, and Table itself is a
// Client Component, so this column configuration lives here, in a client
// module, rather than inline in the (Server Component) decklist page.
// `images` is resolved server-side by the page (see getCardImages) and
// passed down as a plain prop -- this file only renders it.
function CardCategoryTable({
    cards,
    images,
}: {
    cards: Card[];
    images: Record<string, string>;
}) {
    return (
        <Table
            sortable={false}
            tableClassName="table--compact"
            columns={[
                {
                    key: "count",
                    label: "Count",
                    className: "col-count",
                    render: (c: Card) => (
                        <span className="card-count-badge">{c.count}</span>
                    ),
                },
                {
                    key: "card",
                    label: "Card",
                    render: (c: Card) => (
                        <div>
                            <CardHoverPreview
                                imageUrl={images[`${c.set}:${c.number}`]}
                                name={c.name}
                            >
                                <div className="table-title">{c.name}</div>
                            </CardHoverPreview>
                            {c.set ? (
                                <div className="muted tiny">
                                    {c.set}
                                    {c.number ? ` ${c.number}` : ""}
                                </div>
                            ) : null}
                        </div>
                    ),
                },
            ]}
            rows={cards}
        />
    );
}

export function DecklistCategory({
    label,
    cards,
    images,
}: {
    label: string;
    cards: Card[];
    images: Record<string, string>;
}) {
    if (cards.length === 0) return null;

    return (
        <div>
            <p
                className="eyebrow"
                style={{
                    marginBottom: 10,
                    paddingBottom: 8,
                    borderBottom: "1px solid var(--line)",
                }}
            >
                {label} ({cards.reduce((sum, c) => sum + c.count, 0)})
            </p>
            <CardCategoryTable cards={cards} images={images} />
        </div>
    );
}
