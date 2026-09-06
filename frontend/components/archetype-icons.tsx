import { archetypeIconUrl, UNKNOWN_ICON_SLUG } from "@/lib/icons";

// Renders an archetype's icon(s) in place of its name -- fixed dimensions
// avoid layout shift while row images load, and `loading="lazy"` defers
// offscreen rows (e.g. below-the-fold table pages). Falls back to a single
// `unknown` icon instead of a broken image when `icons` is empty/missing
// (no curated icon mapping yet, or no archetype at all).
//
// Uses the native `title` attribute for the hover label (rather than a
// custom tooltip bubble) to match InfoTooltip's convention -- it keeps
// working inside the horizontally-scrolling table wrapper, where an
// absolutely positioned bubble would risk being clipped.
export default function ArchetypeIcons({
    icons,
    name,
    size = 24,
}: {
    icons: string[] | null | undefined;
    name: string;
    size?: number;
}) {
    const slugs = icons && icons.length > 0 ? icons : [UNKNOWN_ICON_SLUG];

    return (
        <span className="archetype-icons" title={name}>
            {slugs.map((slug, i) => (
                <img
                    key={`${slug}-${i}`}
                    src={archetypeIconUrl(slug)}
                    alt={name}
                    width={size}
                    height={size}
                    loading="lazy"
                    className="archetype-icon"
                />
            ))}
        </span>
    );
}
