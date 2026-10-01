import { notFound } from "next/navigation";
import Hero from "@/components/hero";
import Card from "@/components/card";

import { getTournament } from "@/lib/api";
import StandingsTable from "./StandingsTable";
import Link from "next/link";

function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
    }).format(new Date(value));
}

const DIVISION_LABELS: Record<string, string> = {
    MA: "Masters",
    SR: "Seniors",
    JR: "Juniors",
};

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
                {/* breadcrumb */}
                <div style={{ marginBottom: 16 }}>
                    <Link
                        href="/tournaments"
                        className="button"
                        style={{ display: "inline-flex" }}
                    >
                        ← All Events
                    </Link>
                </div>

                <Hero
                    eyebrow="Meta Radar - Tournament Standings"
                    title={tournament.name}
                    lede={
                        <>
                            {formatDate(tournament.date)} ·{" "}
                            {tournament.players.toLocaleString("en-US")} players
                            {tournament.organizer_name
                                ? ` · Hosted by ${tournament.organizer_name}`
                                : ""}
                        </>
                    }
                    meta={
                        <>
                            <span className="pill">{tournament.meta_name}</span>
                            <span
                                className={`badge ${tournament.is_online ? "badge--online" : ""}`}
                            >
                                {tournament.is_online ? "Online" : "In person"}
                            </span>
                        </>
                    }
                />

                {tournament.is_official && tournament.divisions.length > 1 && (
                    <nav
                        aria-label="Divisions"
                        style={{
                            display: "flex",
                            gap: 8,
                            marginBottom: 16,
                            flexWrap: "wrap",
                        }}
                    >
                        {tournament.divisions.map((d) => (
                            <Link
                                key={d.id}
                                href={`/tournaments/${encodeURIComponent(d.id)}`}
                                className="button"
                                aria-current={
                                    d.id === tournament.id ? "page" : undefined
                                }
                                style={
                                    d.id === tournament.id
                                        ? { fontWeight: 700, opacity: 1 }
                                        : { opacity: 0.7 }
                                }
                            >
                                {DIVISION_LABELS[d.division] ?? d.division}
                            </Link>
                        ))}
                    </nav>
                )}

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Leaderboard</p>
                            <h2>Final Standings</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {tournament.standings.length.toLocaleString(
                                "en-US",
                            )}{" "}
                            entries
                        </span>
                    }
                >
                    {tournament.standings.length > 0 ? (
                        <StandingsTable
                            tournamentId={tournament.id}
                            standings={tournament.standings}
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
