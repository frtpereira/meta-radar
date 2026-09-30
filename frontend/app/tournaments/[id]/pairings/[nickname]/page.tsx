import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";
import { getPairings } from "@/lib/api";
import PairingsTable from "./PairingsTable";

type PageParams = { id: string; nickname: string };

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

export default async function TournamentPairingsPage({
    params,
}: {
    params: Promise<PageParams>;
}) {
    const { id, nickname: rawNickname } = await params;
    const nickname = decodeURIComponent(rawNickname);

    const pairings = await getPairings(id, nickname).catch((err: unknown) => {
        if (
            err instanceof Error &&
            err.message.startsWith("Request failed: 404")
        ) {
            notFound();
        }
        throw err;
    });

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <div style={{ marginBottom: 16 }}>
                    <Link
                        href={`/tournaments/${id}`}
                        className="button"
                        style={{ display: "inline-flex" }}
                    >
                        ← Back to {pairings.tournament_name}
                    </Link>
                </div>

                <Hero
                    eyebrow="Meta Radar — Pairings"
                    title={`${pairings.player_name}'s Pairings`}
                    lede={`Round-by-round pairings for ${pairings.player_name} at ${pairings.tournament_name}.`}
                    meta={
                        <>
                            <Link
                                href={`/players/${pairings.tournament_id}`}
                                className="pill"
                            >
                                {pairings.tournament_name}
                            </Link>
                            <Link
                                href={`/players/${encodeURIComponent(pairings.player_name)}`}
                                className="pill"
                            >
                                {pairings.player_name}
                            </Link>
                        </>
                    }
                />

                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Matches</p>
                            <h2>Round-by-Round Pairings</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {pairings.pairings.length.toLocaleString("en-US")}{" "}
                            rounds
                        </span>
                    }
                >
                    {pairings.pairings.length > 0 ? (
                        <PairingsTable pairings={pairings.pairings} />
                    ) : (
                        <EmptyState
                            title="No pairings yet"
                            copy={`${pairings.player_name} has no recorded pairings for this tournament.`}
                        />
                    )}
                </Card>
            </div>
        </main>
    );
}
