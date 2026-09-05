import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
    title: "About | Meta Radar",
    description: "Learn about Meta Radar, a fan-made analytics site for the Pokémon TCG competitive scene.",
};

export default function AboutPage() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />
            <div className="shell">
            <article>
                <h1>About Meta Radar</h1>

                <p>
                    Meta Radar is a fan-made analytics site for the Pokémon Trading Card Game
                    competitive scene. We track tournament results, deck archetypes, and
                    matchup data so players can see what's actually winning — not just what's
                    hyped.
                </p>

                <h2>What we do</h2>

                <p>
                    Every event, standing, and pairing shown on Meta Radar comes from
                    publicly available tournament data on <a href="https://limitlesstcg.com" target="_blank" rel="noopener noreferrer">Limitless TCG</a>,
                    sourced through their official API. We process that data to:
                </p>

                <ul>
                    <li>
                        Group individual decklists into archetypes, separating the "core" cards
                        that define a deck from the flexible slots players tech around.
                    </li>
                    <li>
                        Track how those archetypes perform — win rates, matchup spreads, and
                        how the metagame shifts as new sets and bans land.
                    </li>
                    <li>
                        Give organizers, content creators, and competitive players a faster way
                        to answer "what's actually good right now?" than scrolling through
                        event pages one at a time.
                    </li>
                </ul>

                <h2>What we don't do</h2>

                <p>
                    We don't run tournaments, sell products, or store decklists you haven't
                    already published to a public event. We're not a marketplace, a deck
                    builder, or a substitute for the official rules — just a lens on data
                    that already exists.
                </p>

                <h2>Who's behind it</h2>

                <p>
                    Meta Radar is an independent, fan-built project. It is not developed,
                    operated, or endorsed by The Pokémon Company, Nintendo, Creatures Inc.,
                    Game Freak, or Limitless TCG. Pokémon and all related names, characters,
                    and images are trademarks of their respective owners, used here only to
                    describe the game we're analyzing. See our <Link href="/disclaimers">Disclaimers</Link>{" "}
                    for more on this.
                </p>

                <h2>Get in touch</h2>

                <p>
                    Found a bug, spotted bad data, or have an idea for a feature? Head to our{" "}
                    <Link href="/contact">Contact page</Link>.
                </p>
            </article>
            </div>
        </main>
    );
}
