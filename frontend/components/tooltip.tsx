"use client";

import { useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

// Small floating label matching the site's card styling, shown while
// hovering (or focusing) the entire wrapped element -- not just a "(?)"
// hint icon. Rendered through a portal into <body> and positioned with
// `position: fixed`, so it isn't clipped by ancestors that scroll or clip
// overflow (e.g. the horizontally-scrolling `.table-wrap`), unlike a plain
// CSS-only bubble anchored inside the trigger.
export default function Tooltip({
    label,
    children,
}: {
    label: string;
    children: React.ReactNode;
}) {
    const triggerRef = useRef<HTMLSpanElement>(null);
    const [open, setOpen] = useState(false);
    const [coords, setCoords] = useState({ top: 0, left: 0 });

    useLayoutEffect(() => {
        if (!open || !triggerRef.current) return;
        const rect = triggerRef.current.getBoundingClientRect();
        setCoords({
            top: rect.top,
            left: rect.left + rect.width / 2,
        });
    }, [open]);

    return (
        <span
            ref={triggerRef}
            className="tooltip-trigger"
            tabIndex={0}
            onMouseEnter={() => setOpen(true)}
            onMouseLeave={() => setOpen(false)}
            onFocus={() => setOpen(true)}
            onBlur={() => setOpen(false)}
        >
            {children}
            {open &&
                typeof document !== "undefined" &&
                createPortal(
                    <span
                        role="tooltip"
                        className="tooltip-bubble"
                        style={{ top: coords.top, left: coords.left }}
                    >
                        {label}
                    </span>,
                    document.body,
                )}
        </span>
    );
}
