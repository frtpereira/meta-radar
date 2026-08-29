import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => localStorage.clear());
});

test("homepage renders the dashboard for the default meta", async ({ page }) => {
    await page.goto("/");

    await expect(
        page.getByRole("heading", { name: "Meta Radar Dashboard" }),
    ).toBeVisible();
    await expect(
        page.getByText("Pick a meta to see its live tournaments and archetypes."),
    ).toBeVisible();
    await expect(page.getByText("Worlds 2026").first()).toBeVisible();
    await expect(page.getByText("Worlds Warmup Regional")).toBeVisible();
    await expect(page.getByText("Charizard ex").first()).toBeVisible();
    await expect(
        page.getByText("Ready for the archetype-vs-archetype view."),
    ).toBeVisible();
});

test("homepage handles a meta with no synced data", async ({ page }) => {
    await page.goto("/?meta_id=meta-empty");

    await expect(page.getByText("No tournaments found")).toBeVisible();
    await expect(page.getByText("No archetype stats yet")).toBeVisible();
});
