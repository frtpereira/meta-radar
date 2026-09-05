"use client";

import Link from "next/link";

const FOOTER_LINKS = [
    { href: "/about", label: "About" },
    { href: "/contact", label: "Contact" },
    { href: "/privacy", label: "Privacy Policy" },
    { href: "/tos", label: "Terms of Service" },
    { href: "/disclaimers", label: "Disclaimers" },
];

export function Footer() {
    const currentYear = new Date().getFullYear();

    return (
        <footer className="site-footer">
            <div className="site-footer__inner">
                <nav className="site-footer__links" aria-label="Footer">
                    {FOOTER_LINKS.map(({ href, label }) => (
                        <Link key={href} href={href}>
                            {label}
                        </Link>
                    ))}
                </nav>
                <p className="site-footer__copyright">
                    © {currentYear} Meta Radar
                </p>
            </div>
        </footer>
    );
}
