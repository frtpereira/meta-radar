import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
    title: "Privacy Policy | Meta Radar",
    description: "Meta Radar's privacy policy. Learn how we handle your data.",
};

export default function PrivacyPage() {
    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />
            <div className="shell">
                <article>
                    <h1>Privacy Policy</h1>

                    <p>
                        <strong>Effective date:</strong> <em>[Insert date before publishing]</em>
                    </p>

                    <p>
                        Meta Radar ("Meta Radar," "we," "us," or "our") publishes this Privacy
                        Policy to explain what information we collect when you use
                        metaradar.example (the "Site") and how we handle it. <em>(Replace the
                        placeholder domain and effective date before publishing.)</em>
                    </p>

                    <p>
                        We built Meta Radar to be a lightweight analytics tool, not a platform
                        that needs to know who you are. This policy is short because, by design,
                        we don't collect much.
                    </p>

                    <h2>1. Information we do not collect</h2>

                    <p>
                        Meta Radar does not require an account, login, or registration to use
                        any feature of the Site. We do not knowingly collect:
                    </p>

                    <ul>
                        <li>
                            Names, email addresses, or other contact details (unless you email us
                            directly — see Section 3).
                        </li>
                        <li>Payment information (the Site is free to use).</li>
                        <li>Precise geolocation data.</li>
                    </ul>

                    <h2>2. Information collected automatically</h2>

                    <p>
                        Like most websites, our hosting provider and any analytics tools we use
                        may automatically log standard technical information when you visit,
                        such as:
                    </p>

                    <ul>
                        <li>IP address</li>
                        <li>Browser type and version</li>
                        <li>Device and operating system</li>
                        <li>Pages visited and time spent on the Site</li>
                        <li>Referring website</li>
                    </ul>

                    <p>
                        This information is used only in aggregate, to understand how the Site
                        is used, to diagnose technical problems, and to keep the Site secure. We
                        do not use it to identify individual visitors.
                    </p>

                    <p>
                        <em>(If you add a specific analytics provider, such as Plausible, Google
                        Analytics, or a similar service, name it here along with a link to its
                        own privacy policy and describe any cookies it sets.)</em>
                    </p>

                    <h2>3. Local storage and cookies</h2>

                    <p>
                        The Site uses your browser's local storage to remember your display
                        preference (light or dark theme). This information is stored only on
                        your own device, is never transmitted to our servers, and can be cleared
                        at any time by clearing your browser's site data.
                    </p>

                    <p>
                        We do not currently use tracking cookies or advertising cookies. If that
                        changes, this policy will be updated to describe what's set and why, and
                        a cookie banner will be added where required by law.
                    </p>

                    <h2>4. Information you provide directly</h2>

                    <p>
                        If you contact us by email or open an issue on our GitHub repository, we
                        will see whatever information you choose to include (such as your email
                        address, GitHub username, or the contents of your message). We use this
                        only to respond to you and to improve the Site, and we do not sell or
                        share it with third parties for marketing purposes.
                    </p>

                    <h2>5. Third-party data source</h2>

                    <p>
                        The tournament, player, and match data shown on Meta Radar is sourced
                        from the publicly available Limitless TCG API. Player names and results
                        displayed on the Site reflect information that tournament organizers and
                        Limitless TCG have already made public as part of competitive event
                        results. We do not independently collect this information from players,
                        and requests to correct or remove it may need to be directed to the
                        original event organizer or to Limitless TCG.
                    </p>

                    <h2>6. How we protect information</h2>

                    <p>
                        We take reasonable technical and organizational measures to protect any
                        information described above from unauthorized access, alteration, or
                        disclosure. However, no method of transmission over the internet is
                        completely secure, and we cannot guarantee absolute security.
                    </p>

                    <h2>7. Children's privacy</h2>

                    <p>
                        Meta Radar is not directed at children under 13 (or the minimum age
                        required in your jurisdiction), and we do not knowingly collect personal
                        information from children. If you believe a child has provided us with
                        personal information, please contact us so we can remove it.
                    </p>

                    <h2>8. Your choices</h2>

                    <p>
                        Because we collect so little, there's little to manage — but you can:
                    </p>

                    <ul>
                        <li>
                            Clear your browser's local storage at any time to reset your theme
                            preference.
                        </li>
                        <li>
                            Use browser settings or extensions to block analytics or tracking
                            scripts, if any are added in the future.
                        </li>
                        <li>
                            Contact us to ask what information, if any, we hold that relates to
                            you directly (e.g., from an email you sent us).
                        </li>
                    </ul>

                    <h2>9. Changes to this policy</h2>

                    <p>
                        We may update this Privacy Policy from time to time, for example to
                        reflect a new feature or a new analytics tool. We'll update the
                        effective date at the top of this page when we do. Continued use of the
                        Site after a change means you accept the updated policy.
                    </p>

                    <h2>10. Contact us</h2>

                    <p>
                        Questions about this Privacy Policy? See our <Link href="/contact">Contact page</Link>.
                    </p>
                </article>
            </div>
        </main>
    );
}
