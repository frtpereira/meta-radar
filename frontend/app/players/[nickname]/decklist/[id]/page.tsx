import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";
import { getDecklist } from "@/lib/api";
import { DecklistCategory } from "./DecklistCardsTable";

type PageParams = { nickname: string; id: string };

function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
    }).format(new Date(value));
}

export default async function PlayerDecklistPage({
    params,
}: {
    params: Promise<PageParams>;
}) {
    const { nickname: rawNickname, id } = await params;
    const nickname = decodeURIComponent(rawNickname);

    const decklist = await getDecklist(id).catch((err: unknown) => {
        if (
            err instanceof Error &&
            err.message.startsWith("Request failed: 404")
        ) {
            notFound();
        }
        throw err;
    });

    const byCategory = (cat: string) =>
        decklist.cards.filter((c) => c.category === cat);

    const pokemonCards = byCategory("pokemon");
    const trainerCards = byCategory("trainer");
    const energyCards = byCategory("energy");

    const totalCards = decklist.cards.reduce((sum, c) => sum + c.count, 0);

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <div style={{ marginBottom: 16 }}>
                    <Link
                        href={`/players/${encodeURIComponent(nickname)}`}
                        className="button"
                        style={{ display: "inline-flex" }}
                    >
                        ← Back to {decklist.player_name}
                    </Link>
                </div>

                <Hero
                    eyebrow="Meta Radar — Decklist"
                    title={
                        decklist.archetype_name ??
                        `${decklist.player_name}'s decklist`
                    }
                    lede={`${decklist.player_name}'s decklist from ${decklist.tournament_name} on ${formatDate(decklist.date)}.`}
                    meta={
                        <>
                            <span className="pill">
                                {totalCards} cards
                            </span>
                            <Link
                                href={`/tournaments/${decklist.tournament_id}`}
                                className="pill pill--soft"
                            >
                                {decklist.tournament_name}
                            </Link>
                        </>
                    }
                />

                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Decklist</p>
                            <h2>Exact List</h2>
                        </>
                    }
                >
                    <DecklistCategory label="Pokémon" cards={pokemonCards} />
                    <DecklistCategory label="Trainer" cards={trainerCards} />
                    <DecklistCategory label="Energy" cards={energyCards} />
                </Card>
            </div>
        </main>
    );
}
