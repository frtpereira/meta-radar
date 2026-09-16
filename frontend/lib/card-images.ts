// Card art (see db/migrations/0007_card_images.sql /
// 0008_card_images_language.sql on the backend, and
// scripts/fetch_card_images.py) is self-hosted in a Cloudflare R2
// bucket, exposed at this base URL -- mirrors the icon pattern in
// lib/icons.ts.
const cardBaseUrl =
    process.env.NEXT_PUBLIC_CARD_BASE_URL ??
    "https://cards.metaradar-tcg.com/pokemon";

// Mirrors fetch_card_images.py's VALID_SIZES. XL is the bare filename
// (PBL_081_R_EN.png); every other size appends its own suffix
// (PBL_081_R_EN_MD.png) -- see that script's build_download_url.
export type CardImageSize = "XL" | "LG" | "MD" | "SM" | "XS";

const SIZE_SUFFIXES = new Set<string>(["XL", "LG", "MD", "SM", "XS"]);

// Matches the trailing "_<SIZE>" (if any) plus extension on a filename,
// e.g. "PBL_080_R_EN_MD.png" -> base "PBL_080_R_EN", suffix "MD",
// ext ".png".
const FILENAME_RE = /^(.*?)(?:_([A-Z]{2}))?(\.[a-zA-Z0-9]+)$/;

function resizeFilename(filename: string, size: CardImageSize): string {
    const match = filename.match(FILENAME_RE);
    if (!match) {
        return filename;
    }
    const [, base, existingSuffix, ext] = match;
    // Only strip an existing suffix if it's actually one of our known
    // sizes -- otherwise something like "PBL_080_R_EN.png" (no size
    // suffix, "EN" isn't a size) would get mangled.
    const trueBase =
        existingSuffix && SIZE_SUFFIXES.has(existingSuffix)
            ? base
            : `${base}${existingSuffix ? `_${existingSuffix}` : ""}`;
    return size === "XL" ? `${trueBase}${ext}` : `${trueBase}_${size}${ext}`;
}

// card_images.image_url always comes back from the backend with the MD
// suffix baked in (our sets.txt requests md,sm,xs and MD is the size
// recorded in the DB -- see fetch_card_images.py's db_size), and the
// /card-images endpoint has no notion of size at all. So getting a
// different size means rewriting the filename's suffix ourselves rather
// than asking the backend for a different path.
function resizeUrlOrPath(value: string, size: CardImageSize): string {
    const slash = value.lastIndexOf("/");
    const dir = slash === -1 ? "" : value.slice(0, slash + 1);
    const filename = slash === -1 ? value : value.slice(slash + 1);
    return `${dir}${resizeFilename(filename, size)}`;
}

// card_images.image_url is stored as a bucket-relative path
// ("<SET>/<filename>.png", e.g. "PBL/PBL_080_R_EN_MD.png"), not a full
// URL, so the bucket/CDN can be swapped later without a data migration
// -- this is the one place that turns it into something an <img> can
// load. `size` defaults to MD, matching what's actually stored.
export function cardImageUrl(path: string, size: CardImageSize = "MD") {
    return `${cardBaseUrl}/${resizeUrlOrPath(path, size)}`;
}

// Re-sizes an already-built card image URL (as returned by
// cardImageUrl/getCardImages) without another round trip -- used
// client-side to build a smaller-asset srcSet for mobile from the one
// URL the server already resolved, instead of fetching per size.
export function withCardImageSize(url: string, size: CardImageSize) {
    return resizeUrlOrPath(url, size);
}
