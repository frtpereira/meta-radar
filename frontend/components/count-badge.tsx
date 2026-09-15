type CountBadgeProps = {
    count: number;
    size?: number;
    className?: string;
};

// 1-2 digits fits the circle at the base font size; wider counts (rare,
// but energy lines aren't capped at 4-of the way Pokemon/Trainer are)
// step the font down so the text doesn't spill past the ring.
function fontSizeFor(count: number): number {
    return String(count).length > 2 ? 11 : 15;
}

/**
 * Small circular badge showing a card's copy count, meant to be
 * layered on top of a card image when rendering a decklist export.
 */
export function CountBadge({ count, size = 32, className }: CountBadgeProps) {
    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 32 32"
            xmlns="http://www.w3.org/2000/svg"
            className={className}
            aria-hidden="true"
        >
            <circle
                cx="16"
                cy="16"
                r="14.25"
                fill="#16161f"
                stroke="#d4af37"
                strokeWidth="1.5"
            />
            <circle
                cx="16"
                cy="16"
                r="11.5"
                fill="none"
                stroke="#d4af37"
                strokeWidth="0.5"
                opacity="0.5"
            />
            <text
                x="16"
                y="21"
                textAnchor="middle"
                fontFamily="Arial, sans-serif"
                fontWeight="700"
                fontSize={fontSizeFor(count)}
                fill="#dddddd"
            >
                {count}
            </text>
        </svg>
    );
}

/**
 * Returns the same badge as a standalone SVG markup string, for cases
 * where the export path uses <canvas> / server-side rendering instead
 * of the React tree (e.g. drawing the badge with drawImage via an
 * Image element loaded from a data: URI).
 */
export function countBadgeSvgString(count: number, size = 32): string {
    return `<svg width="${size}" height="${size}" viewBox="0 0 32 32" xmlns="http://www.w3.org/2000/svg">
  <circle cx="16" cy="16" r="14.25" fill="#16161f" stroke="#d4af37" stroke-width="1.5"/>
  <circle cx="16" cy="16" r="11.5" fill="none" stroke="#d4af37" stroke-width="0.5" opacity="0.5"/>
  <text x="16" y="21.5" text-anchor="middle" font-family="Arial, sans-serif" font-weight="700" font-size="${fontSizeFor(count)}" fill="#ffffff">${count}</text>
</svg>`;
}
