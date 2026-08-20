"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const NAV_LINKS = [
    { href: "/", label: "Home" },
    { href: "/tournaments", label: "Tournaments" },
    { href: "/decklists", label: "Decklists" },
    { href: "/matchups", label: "Matchups" },
];

export function NavigationBar() {
    const pathname = usePathname();

    return (
        <header className="site-nav">
            <div className="site-nav__inner">
                <Link className="site-nav__brand" href="/">
                    META Radar
                </Link>
                <nav className="site-nav__links" aria-label="Main">
                    {NAV_LINKS.map(({ href, label }) => {
                        const isCurrent =
                            href === "/"
                                ? pathname === "/"
                                : pathname === href || pathname.startsWith(`${href}/`);

                        return (
                            <Link
                                key={href}
                                href={href}
                                aria-current={isCurrent ? "page" : undefined}
                            >
                                {label}
                            </Link>
                        );
                    })}
                </nav>
            </div>
        </header>
    );
}
