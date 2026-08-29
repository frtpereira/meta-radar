import { expect, test } from "@playwright/test";

// Performance budgets for direct calls against the (mocked) backend API and
// for full page loads through the frontend. These are generous ceilings
// meant to catch severe regressions (e.g. an accidental N+1 query or a
// runaway client-side loop), not to enforce a strict SLA.
const API_RESPONSE_BUDGET_MS = 500;
const PAGE_LOAD_BUDGET_MS = 5000;

const apiBaseUrl = "http://127.0.0.1:4100/api";

const apiEndpoints: { name: string; path: string }[] = [
    { name: "list metas", path: "/metas" },
    { name: "list tournaments", path: "/tournaments?meta_id=meta-2026" },
    { name: "tournament detail", path: "/tournaments/tour-01" },
    { name: "archetype stats", path: "/archetypes/stats?meta_id=meta-2026" },
    { name: "archetype detail", path: "/archetypes/1" },
    { name: "archetype variants", path: "/archetypes/1/variants" },
    { name: "archetype card stats", path: "/archetypes/1/card-stats" },
    { name: "matchup stats", path: "/matchups/stats?meta_id=meta-2026" },
];

for (const endpoint of apiEndpoints) {
    test(`API responds within budget: ${endpoint.name}`, async ({ request }) => {
        const start = Date.now();
        const response = await request.get(`${apiBaseUrl}${endpoint.path}`);
        const elapsed = Date.now() - start;

        expect(response.ok()).toBeTruthy();
        expect(elapsed).toBeLessThan(API_RESPONSE_BUDGET_MS);
    });
}

const pages: { name: string; path: string }[] = [
    { name: "home", path: "/" },
    { name: "tournaments", path: "/tournaments" },
    { name: "matchups", path: "/matchups" },
];

for (const pageDef of pages) {
    test(`page loads within budget: ${pageDef.name}`, async ({ page }) => {
        await page.goto(pageDef.path);
        await page.waitForLoadState("networkidle");

        const navigationTiming = await page.evaluate(() => {
            const [entry] = performance.getEntriesByType(
                "navigation",
            ) as PerformanceNavigationTiming[];
            return entry ? entry.duration : null;
        });

        expect(navigationTiming).not.toBeNull();
        expect(navigationTiming as number).toBeLessThan(PAGE_LOAD_BUDGET_MS);
    });
}
