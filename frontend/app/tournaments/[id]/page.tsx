import Link from "next/link";
import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Table from "@/components/table";
import Card from "@/components/card";

import { getTournament } from "@/lib/api";

function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
    }).format(new Date(value));
}

function formatStanding(value: number) {
    return value === 0 ? "Dropped" : `#${value}`;
}

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

export default async function TournamentPage({
    params,
}: {
    params: Promise<{ id: string }>;
}) {
    const { id } = await params;

    const tournament = await getTournament(id).catch((error: unknown) => {
        if (
            error instanceof Error &&
            error.message.startsWith("Request failed: 404")
        ) {
            notFound();
        }
        throw error;
    });

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="Meta Radar - Tournament Standings"
                    title={tournament.name}
                    lede={
                        <>
                            {formatDate(tournament.date)} ·{" "}
                            {tournament.players.toLocaleString()} players
                            {tournament.organizer_name
                                ? ` · Hosted by ${tournament.organizer_name}`
                                : ""}
                        </>
                    }
                    meta={
                        <>
                            <span className="pill">
                                {tournament.meta_name}
                            </span>
                            <span
                                className={`badge ${tournament.is_online ? "badge--online" : ""}`}
                            >
                                {tournament.is_online ? "Online" : "In person"}
                            </span>
                        </>
                    }
                />

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Leaderboard</p>
                            <h2>Final standings</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {tournament.standings.length.toLocaleString()}{" "}
                            entries
                        </span>
                    }
                >
                    {tournament.standings.length > 0 ? (
                        <Table
                            columns={[
                                {
                                    key: "standing",
                                    label: "Standing",
                                    render: (r: any) =>
                                        formatStanding(r.standing),
                                },
                                {
                                    key: "player",
                                    label: "Player",
                                    render: (r: any) => (
                                        <div className="table-title">
                                            {r.player_name}
                                        </div>
                                    ),
                                },
                                {
                                    key: "archetype",
                                    label: "Archetype",
                                    render: (r: any) =>
                                        r.archetype_name ?? (
                                            <span className="muted tiny">
                                                Unknown
                                            </span>
                                        ),
                                },
                                {
                                    key: "score",
                                    label: "Score",
                                    render: (r: any) =>
                                        `${r.wins}-${r.losses}-${r.ties}`,
                                },
                            ]}
                            rows={tournament.standings}
                        />
                    ) : (
                        <EmptyState
                            title="No standings yet"
                            copy="This tournament hasn't been synced with standings data yet."
                        />
                    )}
                </Card>
            </div>
        </main>
    );
}
