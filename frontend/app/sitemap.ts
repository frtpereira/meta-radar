import type { MetadataRoute } from "next";

export default function sitemap(): MetadataRoute.Sitemap {
    const baseUrl = "https://metaradar-tcg.com";

    // Public pages that should be indexed
    const publicPages = [
        "/",
        "/about",
        "/archetypes",
        "/contact",
        "/matchups",
        "/privacy",
        "/tos",
        "/tournaments",
    ];

    const sitemapEntries: MetadataRoute.Sitemap = publicPages.map((route) => ({
        url: `${baseUrl}${route}`,
        lastModified: new Date(),
        changeFrequency: route === "/" ? "daily" : "weekly",
        priority: route === "/" ? 1.0 : 0.8,
    }));

    return sitemapEntries;
}
