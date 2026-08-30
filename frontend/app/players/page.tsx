import Hero from "@/components/hero";
import Card from "@/components/card";
import PlayerSearch from "./PlayerSearch";

export default function PlayersPage() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="Meta Radar — Players"
                    title="Player Lookup"
                    lede="Search a player's nickname to see their tournament
                        history: placements, archetypes, and decklists."
                />

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Search</p>
                            <h2>Find a Player</h2>
                        </>
                    }
                >
                    <PlayerSearch />
                </Card>
            </div>
        </main>
    );
}
