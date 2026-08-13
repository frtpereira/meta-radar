import type { Metadata } from "next";

// @ts-ignore
import "./globals.css";

export const metadata: Metadata = {
    title: "Pokémon TCG Tracker",
    description: "Track metas, tournaments, and archetype performance.",
};

export default function RootLayout({
    children,
}: Readonly<{
    children: React.ReactNode;
}>) {
    return (
        <html lang="en">
            <body>{children}</body>
        </html>
    );
}
