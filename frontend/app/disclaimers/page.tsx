import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
    title: "Disclaimers | Meta Radar",
    description: "Meta Radar's disclaimer. Learn about limitations and disclaimers.",
};

export default function DisclaimersPage() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />
            <div className="shell">
                <article>
                    <h1>Disclaimers</h1>

                    <h2>Fan site — not an official Pokémon product</h2>

                    <p>
                        Meta Radar is an unofficial, fan-made site created by independent
                        contributors. It is <strong>not</strong> produced, sponsored, endorsed, or approved
                        by The Pokémon Company, The Pokémon Company International, Nintendo,
                        Creatures Inc., Game Freak, or any of their affiliates, nor by Limitless
                        TCG. All Pokémon-related names, terms, characters, card images, and
                        logos are trademarks and/or copyrighted material of their respective
                        owners. Any use of this material on Meta Radar is for identification
                        and commentary purposes only, under fair use, and does not imply any
                        affiliation or endorsement.
                    </p>

                    <h2>Not affiliated with Limitless TCG</h2>

                    <p>
                        Meta Radar consumes publicly available tournament data through the
                        official Limitless TCG API. This does not represent a partnership,
                        endorsement, or official integration with Limitless TCG. Limitless TCG
                        is not responsible for how that data is processed, grouped, or
                        displayed on Meta Radar.
                    </p>

                    <h2>Data accuracy disclaimer</h2>

                    <p>
                        Statistics, archetype groupings, win rates, and trend data on Meta Radar
                        are generated through our own data-processing and clustering logic
                        applied to third-party tournament results. This means:
                    </p>

                    <ul>
                        <li>
                            <strong>Archetypes are analytical groupings, not official categories.</strong>
                            {" "}Two decks with different tech choices may be grouped into the same
                            archetype based on shared "core" cards, which is a judgment call, not
                            a ruling.
                        </li>
                        <li>
                            <strong>Numbers can lag or be revised.</strong> Source data (standings, pairings,
                            drops) is sometimes incomplete, corrected late by organizers, or
                            updated after we've already processed it.
                        </li>
                        <li>
                            <strong>Past performance is not predictive.</strong> Win rates and meta trends
                            describe what has already happened at tracked events — they are not
                            guarantees of how a deck will perform at a future event, and should
                            not be treated as tournament predictions or betting advice.
                        </li>
                    </ul>

                    <p>
                        If you notice data that looks wrong, please tell us — see our{" "}
                        <Link href="/contact">Contact page</Link>.
                    </p>

                    <h2>Not competitive rules guidance</h2>

                    <p>
                        Meta Radar is an analytics tool, not a rules authority. For official
                        tournament rules, deck legality, banned/restricted card lists, or format
                        rotations, always consult the official Pokémon TCG rules resources or
                        your tournament organizer — not Meta Radar's archetype or format
                        labeling.
                    </p>

                    <h2>No professional or financial advice</h2>

                    <p>
                        Nothing on Meta Radar constitutes financial, investment, or purchasing
                        advice regarding singles prices, sealed product, or collectibles. Any
                        mention of card value or scarcity is incidental to competitive analysis,
                        not a recommendation to buy, sell, or trade.
                    </p>

                    <h2>External links</h2>

                    <p>
                        Meta Radar may link to third-party sites, including Limitless TCG,
                        social media profiles, and content creators. We don't control, and
                        aren't responsible for, the content, accuracy, or practices of any
                        linked site.
                    </p>

                    <h2>Limitation of liability</h2>

                    <p>
                        See Section 7 of our <Link href="/tos">Terms of Service</Link> for the
                        full limitation of liability applicable to reliance on Meta Radar's
                        data and content.
                    </p>
                </article>
            </div>
        </main>
    );
}
