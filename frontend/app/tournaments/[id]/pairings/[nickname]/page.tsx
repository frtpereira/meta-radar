import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";
import { getPairings } from "@/lib/api";
import PairingsTable from "./PairingsTable";

type PageParams = { id: string; nickname: string };
// "from" records which page linked here -- the tournament's standings table
// or a player's history table -- so the breadcrumb above can send them back
// the way they came instead of always landing on the tournament. Any other
// (or missing) value falls back to the tournament, since that's the direct
// parent of this route.
type SearchParams = { from?: "player" | "tournament" };

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
    searchParams,
}: {
    params: Promise<PageParams>;
    searchParams: Promise<SearchParams>;
}) {
    const { id, nickname: rawNickname } = await params;
    const nickname = decodeURIComponent(rawNickname);
    const { from } = await searchParams;

    const pairings = await getPairings(id, nickname).catch((err: unknown) => {
        if (
            err instanceof Error &&
            err.message.startsWith("Request failed: 404")
        ) {
            notFound();
        }
        throw err;
    });

    const backHref =
        from === "player"
            ? `/players/${encodeURIComponent(pairings.player_name)}`
            : `/tournaments/${id}`;
    const backLabel =
        from === "player"
            ? `← Back to ${pairings.player_name}`
            : `← Back to ${pairings.tournament_name}`;

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <div style={{ marginBottom: 16 }}>
                    <Link
                        href={backHref}
                        className="button"
                        style={{ display: "inline-flex" }}
                    >
                        {backLabel}
                    </Link>
                </div>

                <Hero
                    eyebrow="Meta Radar — Pairings"
                    title={`${pairings.player_name}'s Pairings`}
                    lede={`Round-by-round pairings for ${pairings.player_name} at ${pairings.tournament_name}.`}
                    meta={
                        <>
                            <Link
                                href={`/tournaments/${pairings.tournament_id}`}
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
