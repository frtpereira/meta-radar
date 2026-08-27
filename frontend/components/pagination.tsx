"use client"

import { useRouter, usePathname, useSearchParams } from "next/navigation";
import React from "react";

export type PageItem = number | "ellipsis-start" | "ellipsis-end";

// Shared by every paginated view (server-driven or client-driven): windows
// the numbered buttons around `page`, but always keeps page 1 and
// `totalPages` visible so long lists don't strand the user without a way
// back to either end -- an "ellipsis-*" entry stands in for the gap.
export function buildPageItems(
    page: number,
    totalPages: number,
    windowSize = 2,
): PageItem[] {
    if (totalPages <= 0) return [];

    const start = Math.max(1, page - windowSize);
    const end = Math.min(totalPages, page + windowSize);

    const items: PageItem[] = [];
    if (start > 1) {
        items.push(1);
        if (start > 2) items.push("ellipsis-start");
    }
    for (let i = start; i <= end; i++) items.push(i);
    if (end < totalPages) {
        if (end < totalPages - 1) items.push("ellipsis-end");
        items.push(totalPages);
    }
    return items;
}

// Presentational prev/numbered/next control, shared between URL-driven
// (Pagination, below) and locally-paginated (Table) usages.
export function PaginationControls({
    page,
    totalPages,
    onPageChange,
}: {
    page: number;
    totalPages: number;
    onPageChange: (page: number) => void;
}) {
    const items = buildPageItems(page, totalPages);

    return (
        <div className="pagination" style={{ marginTop: 12 }}>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <button
                    className="button"
                    onClick={() => onPageChange(Math.max(1, page - 1))}
                    disabled={page <= 1}
                >
                    Prev
                </button>

                {items.map((item, index) =>
                    item === "ellipsis-start" || item === "ellipsis-end" ? (
                        <button
                            key={`${item}-${index}`}
                            className="button button--ellipsis"
                            disabled
                            tabIndex={-1}
                        >
                            …
                        </button>
                    ) : (
                        <button
                            key={item}
                            className={`button ${item === page ? "button--active" : ""}`}
                            onClick={() => onPageChange(item)}
                            aria-current={item === page}
                        >
                            {item}
                        </button>
                    ),
                )}

                <button
                    className="button"
                    onClick={() => onPageChange(Math.min(totalPages, page + 1))}
                    disabled={page >= totalPages}
                >
                    Next
                </button>
            </div>
        </div>
    );
}

export default function Pagination({ page, totalPages }: { page: number; totalPages: number }) {
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();

    function navigateTo(p: number) {
        const sp = new URLSearchParams(searchParams?.toString() ?? "");
        sp.set("page", String(p));
        const url = `${pathname}?${sp.toString()}`;
        // replace to avoid pushing history entries for each click
        router.replace(url);
        // smooth scroll to top for context
        if (typeof window !== "undefined") {
            window.scrollTo({ top: 0, behavior: "smooth" });
        }
    }

    return (
        <PaginationControls page={page} totalPages={totalPages} onPageChange={navigateTo} />
    );
}
