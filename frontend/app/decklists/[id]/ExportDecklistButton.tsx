"use client";

import { useState } from "react";
import { formatDecklistExport } from "@/lib/decklist-export";
import type { Card } from "@/lib/types";

const RESET_DELAY_MS = 2000;

function downloadTextFile(filename: string, contents: string) {
    const blob = new Blob([contents], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
}

export function ExportDecklistButton({
    cards,
    filename = "decklist.txt",
}: {
    cards: Card[];
    filename?: string;
}) {
    const [copied, setCopied] = useState(false);

    async function handleExport() {
        const text = formatDecklistExport(cards);

        // Prefer copying so the list can be pasted straight into a deck
        // builder; fall back to a file download where the Clipboard API
        // isn't available (e.g. insecure contexts, older browsers).
        if (navigator.clipboard?.writeText) {
            try {
                await navigator.clipboard.writeText(text);
                setCopied(true);
                setTimeout(() => setCopied(false), RESET_DELAY_MS);
                return;
            } catch {
                // fall through to download
            }
        }

        downloadTextFile(filename, text);
    }

    return (
        <button
            type="button"
            className="button button--active"
            onClick={handleExport}
        >
            {copied ? "Copied!" : "Export Decklist"}
        </button>
    );
}
