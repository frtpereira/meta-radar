import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import { getCardImages, getDecklist } from "@/lib/api";
import { DecklistView } from "./DecklistView";

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

    // Resolved server-side, alongside the decklist itself, and handed down
    // as a plain prop -- one batched request for the whole page instead of
    // three (one per category) firing client-side after hydration, and no
    // "preview not ready yet" flash on first hover.
    const images = await getCardImages(decklist.cards).catch(() => ({}));

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
                            <Link
                                href={`/players/${decklist.player_name}`}
                                className="pill"
                            >
                                {decklist.player_name}
                            </Link>
                            <Link
                                href={`/tournaments/${decklist.tournament_id}`}
                                className="pill pill--soft"
                            >
                                {decklist.tournament_name}
                            </Link>
                        </>
                    }
                />

                <DecklistView
                    cards={decklist.cards}
                    images={images}
                    filename={`${decklist.player_name}-decklist.txt`}
                />
            </div>
        </main>
    );
}
