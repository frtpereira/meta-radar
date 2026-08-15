import Link from "next/link";
import { notFound } from "next/navigation";

import { getTournament } from "@/lib/api";

function formatDate(value: string) {
    return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
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
        if (error instanceof Error && error.message.startsWith("Request failed: 404")) {
            notFound();
        }
        throw error;
    });

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <header className="hero card">
                    <div>
                        <Link className="pill pill--soft" href="/">
                            ← Back to dashboard
                        </Link>
                        <p className="eyebrow" style={{ marginTop: 16 }}>
                            {tournament.format_code}
                        </p>
                        <h1>{tournament.name}</h1>
                        <p className="lede">
                            {formatDate(tournament.date)} ·{" "}
                            {tournament.players.toLocaleString()} players
                            {tournament.organizer_name
                                ? ` · Hosted by ${tournament.organizer_name}`
                                : ""}
                        </p>
                    </div>

                    <div className="hero__meta">
                        <span
                            className={`badge ${tournament.is_online ? "badge--online" : ""}`}
                        >
                            {tournament.is_online ? "Online" : "In person"}
                        </span>
                    </div>
                </header>

                <section className="card section">
                    <div className="section__heading">
                        <div>
                            <p className="eyebrow">Leaderboard</p>
                            <h2>Final standings</h2>
                        </div>
                        <span className="muted">
                            {tournament.standings.length.toLocaleString()} entries
                        </span>
                    </div>

                    {tournament.standings.length > 0 ? (
                        <div className="table-wrap">
                            <table>
                                <thead>
                                    <tr>
                                        <th>Standing</th>
                                        <th>Player</th>
                                        <th>Archetype</th>
                                        <th>Score</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {tournament.standings.map((row) => (
                                        <tr key={row.player_id}>
                                            <td>{formatStanding(row.standing)}</td>
                                            <td>
                                                <div className="table-title">
                                                    {row.player_name}
                                                </div>
                                            </td>
                                            <td>
                                                {row.archetype_name ?? (
                                                    <span className="muted tiny">
                                                        Unknown
                                                    </span>
                                                )}
                                            </td>
                                            <td>
                                                {row.wins}-{row.losses}
                                                {row.ties > 0 ? `-${row.ties}` : ""}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    ) : (
                        <EmptyState
                            title="No standings yet"
                            copy="This tournament hasn't been synced with standings data yet."
                        />
                    )}
                </section>
            </div>
        </main>
    );
}
