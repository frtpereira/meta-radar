"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ThemeToggle } from "@/components/theme-toggle";

const NAV_LINKS = [
    { href: "/", label: "Home" },
    { href: "/tournaments", label: "Events" },
    { href: "/archetypes", label: "Archetypes" },
    { href: "/matchups", label: "Matchups" },
    { href: "/players", label: "Players" },
];

export function NavigationBar() {
    const pathname = usePathname();
    const [isMenuOpen, setIsMenuOpen] = useState(false);

    // Close menu when pathname changes
    useEffect(() => {
        setIsMenuOpen(false);
    }, [pathname]);

    // Close menu when clicking outside (on body)
    useEffect(() => {
        if (!isMenuOpen) return;

        const handleClickOutside = (event: MouseEvent) => {
            const nav = document.querySelector(".site-nav");
            if (nav && !nav.contains(event.target as Node)) {
                setIsMenuOpen(false);
            }
        };

        document.addEventListener("click", handleClickOutside);
        return () => document.removeEventListener("click", handleClickOutside);
    }, [isMenuOpen]);

    const toggleMenu = () => {
        setIsMenuOpen(!isMenuOpen);
    };

    return (
        <header className="site-nav">
            <div className="site-nav__inner">
                <Link className="site-nav__brand" href="/">
                    META Radar
                </Link>
                <button
                    className="site-nav__menu-button"
                    onClick={toggleMenu}
                    aria-label="Toggle navigation menu"
                    aria-expanded={isMenuOpen}
                >
                    <span></span>
                    <span></span>
                    <span></span>
                </button>
                <div
                    className={`site-nav__right ${
                        isMenuOpen ? "site-nav__right--open" : ""
                    }`}
                >
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
