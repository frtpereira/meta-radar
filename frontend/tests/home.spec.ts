import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => localStorage.clear());
});

test("homepage renders the dashboard for the default meta", async ({ page }) => {
    await page.goto("/");

    await expect(
        page.getByRole("heading", { name: "Meta Radar Dashboard" }),
    ).toBeVisible();
    await expect(page.getByText("Worlds 2026").first()).toBeVisible();
    await expect(page.getByText("Worlds Warmup Regional")).toBeVisible();
    await expect(page.getByText("Charizard ex").first()).toBeVisible();

    await expect(page.getByText("Top win rate deck")).toBeVisible();
    await expect(page.getByText("Top played deck")).toBeVisible();
    await expect(page.getByText("Latest Doom tournament")).toBeVisible();
    await expect(page.getByText("Doom's Local Cup")).toBeVisible();
    await expect(page.getByText("Next set release")).toBeVisible();
});

test("homepage handles a meta with no synced data", async ({ page }) => {
    await page.goto("/?meta_id=meta-empty");

    await expect(page.getByText("No tournaments found")).toBeVisible();
    await expect(page.getByText("No archetype stats yet")).toBeVisible();
});
