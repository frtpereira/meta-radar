"use client";

import { usePathname, useRouter } from "next/navigation";
import type { FormEvent, ReactNode } from "react";

// Plain HTML `method="get"` forms serialize every field, including ones the
// user left blank or at their neutral default (e.g. "0" for a "minimum"
// input) -- so submitting a filter form with only a couple of fields set
// still produces a URL cluttered with `field=` for everything else. This
// wrapper intercepts submit and only keeps fields that carry an actual
// filter value, then navigates with that trimmed-down query string.
export default function FilterForm({
    children,
    className,
    style,
}: {
    children: ReactNode;
    className?: string;
    style?: React.CSSProperties;
}) {
    const router = useRouter();
    const pathname = usePathname();

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();

        const form = event.currentTarget;
        const formData = new FormData(form);
        const params = new URLSearchParams();

        for (const [name, rawValue] of formData.entries()) {
            if (typeof rawValue !== "string") continue;

            const value = rawValue.trim();
            if (!value) continue;

            const field = form.elements.namedItem(name);
            // A number input left at "0" (e.g. "minimum players") means "no
            // minimum" -- functionally empty, so it shouldn't appear either.
            if (
                field instanceof HTMLInputElement &&
                field.type === "number" &&
                Number(value) === 0
            ) {
                continue;
            }

            params.set(name, value);
        }

        const query = params.toString();
        router.push(query ? `${pathname}?${query}` : pathname);
    }

    return (
        <form onSubmit={handleSubmit} className={className} style={style}>
            {children}
        </form>
    );
}
