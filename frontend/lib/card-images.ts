// Card art (see db/migrations/0007_card_images.sql /
// 0008_card_images_language.sql on the backend, and
// scripts/fetch_card_images.py) is self-hosted in a Cloudflare R2
// bucket, exposed at this base URL -- mirrors the icon pattern in
// lib/icons.ts.
const cardBaseUrl =
    process.env.NEXT_PUBLIC_CARD_BASE_URL ??
    "https://cards.metaradar-tcg.com/pokemon";

// card_images.image_url is stored as a bucket-relative path
// ("<SET>/<filename>.png", e.g. "PBL/PBL_080_R_EN.png"), not a full URL,
// so the bucket/CDN can be swapped later without a data migration --
// this is the one place that turns it into something an <img> can load.
export function cardImageUrl(path: string) {
    return `${cardBaseUrl}/${path}`;
}
