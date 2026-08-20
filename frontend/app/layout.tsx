import type { Metadata } from "next";
import { NavigationBar } from "@/components/navigation-bar";

import "./globals.css";

export const metadata: Metadata = {
    title: "Pokémon TCG Tracker",
    description: "Track metas, tournaments, and archetype performance.",
};

// Runs before first paint so the page never flashes the wrong theme.
// Honors a manually-picked theme (persisted in localStorage) if one
// exists; otherwise defaults to the OS-level `prefers-color-scheme`.
const THEME_INIT_SCRIPT = `
(function () {
  try {
    var stored = localStorage.getItem("theme");
    var theme =
      stored === "light" || stored === "dark"
        ? stored
        : window.matchMedia("(prefers-color-scheme: light)").matches
          ? "light"
          : "dark";
    document.documentElement.setAttribute("data-theme", theme);
  } catch (e) {}
})();
`;

export default function RootLayout({
    children,
}: Readonly<{
    children: React.ReactNode;
}>) {
    return (
        <html lang="en" suppressHydrationWarning>
            <head>
                <script
                    dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }}
                />
            </head>
            <body>
                <NavigationBar />
                {children}
            </body>
        </html>
    );
}
