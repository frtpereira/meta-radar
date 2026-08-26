import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => localStorage.clear());
});

test("matchups page renders stats and supports filter interactions", async ({ page }) => {
    await page.goto("/matchups");

    await expect(page.getByRole("heading", { name: "Matchup Analysis" })).toBeVisible();
    await page.getByLabel("Filter by archetype").selectOption("1");
    await page.getByRole("button", { name: "Load matchups" }).click();

    await expect(page).toHaveURL(/archetype_id=1/);
    await expect(page.getByRole("heading", { name: "Matchup Analysis" })).toBeVisible();
    await expect(page.getByRole("table").getByText("Gardevoir ex")).toBeVisible();
    await expect(page.getByRole("table").getByText("14-8-2")).toBeVisible();
});

test("matchups page shows empty and server-error fallback states", async ({ page }) => {
    await page.goto("/matchups?meta_id=meta-empty");
    await expect(page.getByText("No matchup stats found")).toBeVisible();

    await page.goto("/matchups?meta_id=meta-error");
    await expect(page.getByText("No matchup stats found")).toBeVisible();
});
