import type { Metadata } from "next";
import { NavigationBar } from "@/components/navigation-bar";

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
            <body>
                <NavigationBar />
                {children}
            </body>
        </html>
    );
}
