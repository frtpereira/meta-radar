"use client"

import { useRouter, usePathname, useSearchParams } from "next/navigation";
import React from "react";

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

    const windowSize = 2; // show current +/- 2
    const pages: number[] = [];
    for (let i = Math.max(1, page - windowSize); i <= Math.min(totalPages, page + windowSize); i++) {
        pages.push(i);
    }

    return (
        <div className="pagination" style={{ marginTop: 12 }}>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <button
                    className="button"
                    onClick={() => navigateTo(Math.max(1, page - 1))}
                    disabled={page <= 1}
                >
                    Prev
                </button>

                {pages.map((p) => (
                    <button
                        key={p}
                        className={`button ${p === page ? "button--active" : ""}`}
                        onClick={() => navigateTo(p)}
                        aria-current={p === page}
                    >
                        {p}
                    </button>
                ))}

                <button
                    className="button"
                    onClick={() => navigateTo(Math.min(totalPages, page + 1))}
                    disabled={page >= totalPages}
                >
                    Next
                </button>
            </div>
        </div>
    );
}
