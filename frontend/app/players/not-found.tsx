import Link from "next/link";
import Hero from "@/components/hero";
import Card from "@/components/card";

export default function PlayerNotFound() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <div style={{ marginTop: 16 }}>
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
                    title="Player not found"
                    lede="We couldn't find a player, decklist, or nickname
                        matching that URL."
                />
            </div>
        </main>
    );
}
