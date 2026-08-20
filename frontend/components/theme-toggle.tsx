"use client";

import { useEffect, useState } from "react";

type Theme = "light" | "dark";

const STORAGE_KEY = "theme";

function systemTheme(): Theme {
    if (typeof window === "undefined") return "dark";
    return window.matchMedia("(prefers-color-scheme: light)").matches
        ? "light"
        : "dark";
}

function SunIcon() {
    return (
        <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
        >
            <circle cx="12" cy="12" r="4" />
            <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
        </svg>
    );
}

function MoonIcon() {
    return (
        <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
        >
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
    );
}

// Reads the theme the blocking init script (see frontend/app/layout.tsx) already
// applied to <html data-theme="...">, so the button never has to guess or
// flash between states — it just mirrors and updates that attribute.
export function ThemeToggle() {
    const [theme, setTheme] = useState<Theme>("dark");
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
        setTheme(
            (document.documentElement.getAttribute("data-theme") as Theme) ||
                systemTheme(),
        );

        // Live-follow the OS setting for as long as the user hasn't made an
        // explicit choice of their own.
        const media = window.matchMedia("(prefers-color-scheme: light)");
        const onSystemChange = () => {
            if (localStorage.getItem(STORAGE_KEY)) return;
            const next = systemTheme();
            document.documentElement.setAttribute("data-theme", next);
            setTheme(next);
        };
        media.addEventListener("change", onSystemChange);
        return () => media.removeEventListener("change", onSystemChange);
    }, []);

    function toggleTheme() {
        const next: Theme = theme === "light" ? "dark" : "light";
        document.documentElement.setAttribute("data-theme", next);
        localStorage.setItem(STORAGE_KEY, next);
        setTheme(next);
    }

    return (
        <button
            type="button"
            className="theme-toggle"
            onClick={toggleTheme}
            aria-label={
                theme === "light"
                    ? "Switch to dark theme"
                    : "Switch to light theme"
            }
            title={
                theme === "light"
                    ? "Switch to dark theme"
                    : "Switch to light theme"
            }
        >
            {/* Avoid rendering the "wrong" icon before mount corrects it */}
            {mounted && theme === "light" ? <SunIcon /> : <MoonIcon />}
        </button>
    );
}
