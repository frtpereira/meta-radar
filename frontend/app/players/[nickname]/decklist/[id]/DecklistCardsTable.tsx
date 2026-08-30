"use client";

import Table from "@/components/table";
import type { Card } from "@/lib/types";

// Table `columns` entries carry `render` functions, and Table itself is a
// Client Component, so this column configuration lives here, in a client
// module, rather than inline in the (Server Component) decklist page.
function CardCategoryTable({ cards }: { cards: Card[] }) {
    return (
        <Table
            sortable={false}
            columns={[
                {
                    key: "count",
                    label: "Count",
                    render: (c: Card) => (
                        <span
                            style={{
                                display: "inline-flex",
                                alignItems: "center",
                                justifyContent: "center",
                                width: 28,
                                height: 28,
                                borderRadius: "50%",
                                background: "rgba(255,209,102,0.12)",
                                border: "1px solid rgba(255,209,102,0.3)",
                                color: "var(--accent)",
                                fontWeight: 700,
                                fontSize: "0.85rem",
                                fontFamily: "Georgia, serif",
                            }}
                        >
                            {c.count}
                        </span>
                    ),
                },
                {
                    key: "card",
                    label: "Card",
                    render: (c: Card) => (
                        <div>
                            <div className="table-title">{c.name}</div>
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
}: {
    label: string;
    cards: Card[];
}) {
    if (cards.length === 0) return null;

    return (
        <div style={{ marginTop: 24 }}>
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
            <CardCategoryTable cards={cards} />
        </div>
    );
}
