import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
    title: "Terms of Service | Meta Radar",
    description:
        "Meta Radar's Terms of Service. Read the terms governing your use of our site.",
};

export default function TermsOfServicePage() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />
            <div className="shell">
                <article>
                    <h1>Terms of Service</h1>

                    <p>
                        <strong>Effective date:</strong> Tuesday, 1 September
                        2026
                    </p>

                    <p>
                        Welcome to Meta Radar. These Terms of Service ("Terms")
                        govern your use of metaradar-tcg.com (the "Site"). By
                        accessing or using the Site, you agree to these Terms.
                        If you don't agree, please don't use the Site.
                    </p>

                    <h2>1. Who we are</h2>

                    <p>
                        Meta Radar is an independent, fan-operated analytics
                        site for the Pokémon Trading Card Game competitive
                        scene. It is not operated, sponsored, or endorsed by The
                        Pokémon Company, Nintendo, Creatures Inc., Game Freak,
                        or Limitless TCG. See our{" "}
                        <Link href="/disclaimers">Disclaimers</Link> for more.
                    </p>

                    <h2>2. Use of the Site</h2>

                    <p>
                        You may use Meta Radar for personal, non-commercial
                        purposes, such as research, content creation, or
                        preparing for competitive play. You agree not to:
                    </p>

                    <ul>
                        <li>
                            Use automated means (scraping, bots, crawlers) to
                            extract data from the Site at a volume or frequency
                            that could disrupt its operation, or in a way that
                            circumvents the underlying Limitless TCG API's own
                            terms of use.
                        </li>
                        <li>
                            Attempt to gain unauthorized access to the Site's
                            systems or data.
                        </li>
                        <li>
                            Use the Site to violate any applicable law, or to
                            infringe the rights of any third party.
                        </li>
                        <li>
                            Misrepresent Meta Radar's data as official
                            tournament results, rankings, or statements from The
                            Pokémon Company, Nintendo, or Limitless TCG.
                        </li>
                    </ul>

                    <p>
                        We reserve the right to restrict or block access for
                        anyone who violates these Terms.
                    </p>

                    <h2>3. Data accuracy</h2>

                    <p>
                        Meta Radar's tournament, archetype, and matchup
                        statistics are derived from third-party data (primarily
                        the Limitless TCG API) and from our own processing logic
                        (such as archetype clustering and win-rate
                        calculations). We make reasonable efforts to keep this
                        data accurate and up to date, but:
                    </p>

                    <ul>
                        <li>
                            We do not guarantee the completeness, accuracy, or
                            timeliness of any data shown on the Site.
                        </li>
                        <li>
                            Archetype groupings, trends, and statistics are
                            analytical judgments, not official rulings or
                            classifications.
                        </li>
                        <li>
                            Data may be delayed, incomplete, or corrected after
                            publication as source data changes.
                        </li>
                    </ul>

                    <p>
                        See our <Link href="/disclaimers">Disclaimers</Link> for
                        the full disclaimer of warranties around data accuracy.
                    </p>

                    <h2>4. Intellectual property</h2>

                    <h3>Our content</h3>

                    <p>
                        The design, code, text, and original analysis on Meta
                        Radar (excluding third-party trademarks and underlying
                        tournament data) are owned by Meta Radar or its
                        contributors. You may reference or link to our pages,
                        including for content creation, with attribution. You
                        may not copy or republish substantial portions of our
                        analysis, design, or codebase as your own without
                        permission.
                    </p>

                    <h3>Third-party trademarks and content</h3>

                    <p>
                        Pokémon, Pokémon TCG, and all associated names, logos,
                        card images, and character designs are trademarks and
                        copyrighted material of their respective owners,
                        including The Pokémon Company, Nintendo, Creatures Inc.,
                        and Game Freak. Meta Radar uses these names and
                        references only descriptively, under fair use, to
                        identify the game and cards being discussed. We claim no
                        ownership over this material.
                    </p>

                    <p>
                        Tournament and player data displayed on the Site
                        originates from Limitless TCG and the organizers of the
                        underlying events; we display it under the terms of
                        Limitless TCG's public API.
                    </p>

                    <h2>5. Third-party links</h2>

                    <p>
                        The Site may link to third-party sites (such as
                        Limitless TCG, social media, or content creators). We
                        don't control these sites and aren't responsible for
                        their content, policies, or practices. Visiting them is
                        at your own risk.
                    </p>

                    <h2>6. Disclaimer of warranties</h2>

                    <p>
                        The Site and all its content are provided "as is" and
                        "as available," without warranties of any kind, express
                        or implied, including implied warranties of
                        merchantability, fitness for a particular purpose, or
                        non-infringement. We do not warrant that the Site will
                        be uninterrupted, error-free, or secure.
                    </p>

                    <h2>7. Limitation of liability</h2>

                    <p>
                        To the fullest extent permitted by law, Meta Radar and
                        its contributors will not be liable for any indirect,
                        incidental, special, consequential, or punitive damages,
                        or any loss of data, revenue, or opportunity, arising
                        from your use of (or inability to use) the Site —
                        including reliance on any statistic, archetype grouping,
                        or matchup data shown on it, such as in preparing for a
                        competitive event.
                    </p>

                    <h2>8. Changes to the Site or these Terms</h2>

                    <p>
                        We may modify, suspend, or discontinue any part of the
                        Site at any time. We may also update these Terms from
                        time to time; we'll update the effective date above when
                        we do. Continued use of the Site after changes take
                        effect means you accept the revised Terms.
                    </p>

                    <h2>9. Governing law</h2>

                    <p>
                        These Terms are governed by the laws of Portugal,
                        without regard to conflict-of-law principles.
                    </p>

                    <h2>10. Contact</h2>

                    <p>
                        Questions about these Terms? See our{" "}
                        <Link href="/contact">Contact page</Link>.
                    </p>
                </article>
            </div>
        </main>
    );
}
