"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

// Offset from the cursor, and the preview's rendered dimensions -- kept
// in sync with the `.card-preview-image` width (and its 5/7 aspect
// ratio) in globals.css so the edge overflow checks below are accurate.
const OFFSET_X = 20;
const OFFSET_Y = 20;
const PREVIEW_WIDTH = 220;
const PREVIEW_HEIGHT = (PREVIEW_WIDTH * 7) / 5;

// Card art preview that follows the cursor while hovering `children`
// (typically a card name in a table row). Modeled on
// components/tooltip.tsx -- same portal-into-<body> + `position: fixed`
// approach so it isn't clipped by the horizontally-scrolling table
// wrapper -- but repositions on every `mousemove` instead of anchoring to
// the trigger element, since the point here is a preview that tracks the
// cursor rather than a label pinned above/below it.
//
// `imageUrl` is resolved up front by the caller (see getCardImages in
// lib/api.ts, called server-side by the page and passed down as a
// prop), so a whole table's worth of hovers share one batched fetch
// instead of firing a request per hover.
export default function CardHoverPreview({
    imageUrl,
    name,
    children,
}: {
    imageUrl?: string;
    name?: string;
    children: React.ReactNode;
}) {
    const [open, setOpen] = useState(false);
    const [coords, setCoords] = useState({ top: 0, left: 0 });
    const frame = useRef<number | null>(null);

    useEffect(() => {
        return () => {
            if (frame.current !== null) cancelAnimationFrame(frame.current);
        };
    }, []);

    if (!imageUrl) {
        // No art resolved for this card (yet, or ever) -- render plain
        // rather than a hover target that does nothing.
        return <>{children}</>;
    }

    const updateCoords = (e: React.MouseEvent) => {
        // Flip to the left of the cursor once the preview would overflow
        // the right edge of the viewport.
        const overflowsRight =
            e.clientX + OFFSET_X + PREVIEW_WIDTH > window.innerWidth;
        const left = overflowsRight
            ? e.clientX - OFFSET_X - PREVIEW_WIDTH
            : e.clientX + OFFSET_X;

        // Flip above the cursor once the preview would overflow the
        // bottom edge of the viewport.
        const overflowsBottom =
            e.clientY + OFFSET_Y + PREVIEW_HEIGHT > window.innerHeight;
        const top = overflowsBottom
            ? e.clientY - OFFSET_Y - PREVIEW_HEIGHT
            : e.clientY + OFFSET_Y;

        // rAF-throttle so a burst of mousemove events doesn't queue up more
        // state updates than the browser can actually paint.
        if (frame.current !== null) return;
        frame.current = requestAnimationFrame(() => {
            setCoords({ top, left });
            frame.current = null;
        });
    };

    return (
        <span
            className="card-preview-trigger"
            onMouseEnter={(e) => {
                updateCoords(e);
                setOpen(true);
            }}
            onMouseMove={updateCoords}
            onMouseLeave={() => setOpen(false)}
        >
            {children}
            {open &&
                typeof document !== "undefined" &&
                createPortal(
                    <img
                        src={imageUrl}
                        alt={name ?? ""}
                        className="card-preview-image"
                        style={{ top: coords.top, left: coords.left }}
                    />,
                    document.body,
                )}
        </span>
    );
}
