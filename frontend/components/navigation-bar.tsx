"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ThemeToggle } from "@/components/theme-toggle";

const NAV_LINKS = [
    { href: "/", label: "Home" },
    { href: "/tournaments", label: "Events" },
    { href: "/decklists", label: "Decks" },
    { href: "/matchups", label: "Matchups" },
    { href: "/players", label: "Players" },
];

export function NavigationBar() {
    const pathname = usePathname();

    return (
        <header className="site-nav">
            <div className="site-nav__inner">
                <Link className="site-nav__brand" href="/">
                    META Radar
                </Link>
                <div className="site-nav__right">
                    <nav className="site-nav__links" aria-label="Main">
                        {NAV_LINKS.map(({ href, label }) => {
                            const isCurrent =
                                href === "/"
                                    ? pathname === "/"
                                    : pathname === href ||
                                      pathname.startsWith(`${href}/`);

                            return (
                                <Link
                                    key={href}
                                    href={href}
                                    aria-current={
                                        isCurrent ? "page" : undefined
                                    }
                                >
                                    {label}
                                </Link>
                            );
                        })}
                    </nav>
                    <ThemeToggle />
                </div>
            </div>
        </header>
    );
}
