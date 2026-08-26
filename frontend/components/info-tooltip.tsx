import React from "react";

// Small "(?)" indicator that shows explanatory text on hover/focus.
// Uses the native `title` attribute (rather than a custom CSS bubble) so it
// still works correctly inside containers with `overflow: auto` — such as
// the horizontally-scrolling table wrapper — where an absolutely positioned
// bubble would risk being clipped.
export default function InfoTooltip({ text }: { text: string }) {
    return (
        <span
            className="info-tooltip"
            tabIndex={0}
            title={text}
            // Table headers can wrap this in a sort button; keep taps/clicks
            // on the tooltip from also triggering a sort toggle.
            onClick={(e) => e.stopPropagation()}
        >
            ?<span className="sr-only"> {text}</span>
        </span>
    );
}
