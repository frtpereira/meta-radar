import type { Metadata } from "next";

export const metadata: Metadata = {
    title: "Contact | Meta Radar",
    description:
        "Get in touch with the Meta Radar team. Report bugs, request features, or send feedback.",
};

export default function ContactPage() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />
            <div className="shell">
                <article>
                    <h1>Contact</h1>

                    <p>
                        We'd rather hear from you than have you sit on a bug or
                        a bad take about a matchup number.
                    </p>

                    <h2>Bug reports & data issues</h2>

                    <p>
                        If something looks wrong — a mislabeled archetype, a
                        missing tournament, a stat that doesn't add up — the
                        fastest way to get it fixed is to open an issue on our
                        GitHub repository:
                    </p>

                    <ul>
                        <li>
                            <strong>GitHub Issues:</strong>{" "}
                            <a
                                href="https://github.com/frtpereira/meta-radar/issues"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                github.com/frtpereira/meta-radar/issues
                            </a>
                        </li>
                    </ul>

                    <p>
                        Please include the page URL, the event or archetype in
                        question, and (if you can) a screenshot. It helps a lot.
                    </p>

                    <h2>Feature requests & feedback</h2>

                    <p>
                        Same place — open an issue and tag it as a feature
                        request. We read all of them, even if we can't reply to
                        each one individually.
                    </p>

                    <h2>Everything else</h2>

                    <p>
                        For privacy questions, takedown requests, legal notices,
                        or anything that doesn't fit a GitHub issue, email us
                        at:
                    </p>

                    <p>
                        <strong>
                            <a href="mailto:contact@metaradar-tcg.com">
                                contact@metaradar.example
                            </a>
                        </strong>
                    </p>

                    <p>
                        We aim to respond to direct emails within a few business
                        days. Response times may be longer for GitHub issues,
                        since those are handled as time allows by a small,
                        independent team.
                    </p>

                    <h2>Data source</h2>

                    <p>
                        Meta Radar sources tournament data from the{" "}
                        <a
                            href="https://limitlesstcg.com"
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Limitless TCG
                        </a>{" "}
                        API. If your concern is about the underlying tournament
                        data itself (rather than how we display or analyze it),
                        you may also want to reach out to Limitless TCG
                        directly.
                    </p>
                </article>
            </div>
        </main>
    );
}
