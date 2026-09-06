import { archetypeIconUrl, UNKNOWN_ICON_SLUG } from "@/lib/icons";
import Tooltip from "@/components/tooltip";

// Renders an archetype's icon(s) in place of its name -- fixed dimensions
// avoid layout shift while row images load, and `loading="lazy"` defers
// offscreen rows (e.g. below-the-fold table pages). Falls back to a single
// `unknown` icon instead of a broken image when `icons` is empty/missing
// (no curated icon mapping yet, or no archetype at all).
//
// Wrapped in `Tooltip` so hovering (or focusing) anywhere across the whole
// icon group -- not just a single icon -- shows the deck name in a bubble
// styled to match the rest of the site.
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
        <Tooltip label={name}>
            <span className="archetype-icons">
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
        </Tooltip>
    );
}
