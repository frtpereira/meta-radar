import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";
import { getPlayer } from "@/lib/api";
import PlayerHistoryTable from "./PlayerHistoryTable";

type PageParams = { nickname: string };

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

export default async function PlayerDetailPage({
    params,
}: {
    params: Promise<PageParams>;
}) {
    const { nickname: rawNickname } = await params;
    const nickname = decodeURIComponent(rawNickname);

    const player = await getPlayer(nickname).catch((err: unknown) => {
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
                        href="/players"
                        className="button"
                        style={{ display: "inline-flex" }}
                    >
                        ← Search another player
                    </Link>
                </div>

                <Hero
                    eyebrow="Meta Radar — Player"
                    title={player.name}
                    lede={`Tournament history for ${player.name}.`}
                    meta={
                        <span className="pill">
                            {player.history.length.toLocaleString()} results
                        </span>
                    }
                />

                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">History</p>
                            <h2>Tournament Results</h2>
                        </>
                    }
                >
                    {player.history.length > 0 ? (
                        <PlayerHistoryTable
                            nickname={player.name}
                            history={player.history}
                        />
                    ) : (
                        <EmptyState
                            title="No tournament history"
                            copy={`${player.name} hasn't played in any tracked tournaments yet.`}
                        />
                    )}
                </Card>
            </div>
        </main>
    );
}
