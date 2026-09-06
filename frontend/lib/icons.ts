// Pokémon icons (see db/migrations/0006_pokemon_icons.sql on the backend)
// are self-hosted in a Cloudflare R2 bucket, exposed at this base URL --
// mirrors the NEXT_PUBLIC_API_BASE_URL pattern in lib/api.ts.
const iconBaseUrl =
    process.env.NEXT_PUBLIC_ICON_BASE_URL ??
    "https://icons.metaradar-tcg.com/pokemon-icons";

// Rendered whenever an archetype has no icon slugs -- either it has no
// curated icon mapping yet, or (for standings/decklists) no archetype was
// identified at all -- so a broken image is never shown.
export const UNKNOWN_ICON_SLUG = "unknown";

export function archetypeIconUrl(slug: string) {
    return `${iconBaseUrl}/${slug}.png`;
}
