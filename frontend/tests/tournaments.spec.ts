import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => localStorage.clear());
});

test("tournaments page renders results and paginates between pages", async ({ page }) => {
    await page.goto("/tournaments");

    const pagination = page.locator(".pagination").first();

    await expect(page.getByRole("heading", { name: "Tournament Explorer" })).toBeVisible();
    await expect(page.getByText("Page 1 of 2")).toBeVisible();
    await expect(page.getByText("Worlds Warmup Regional")).toBeVisible();
    await expect(page.getByText("Tournament 21")).toHaveCount(0);

    await pagination.getByRole("button", { name: /^Next$/ }).click();
    await expect(page).toHaveURL(/page=2/);
    await expect(page.getByText("Page 2 of 2")).toBeVisible();
    await expect(page.getByText("Tournament 21")).toBeVisible();

    await pagination.getByRole("button", { name: /^Prev$/ }).click();
    await expect(page).toHaveURL(/\/tournaments(\?|$)/);
    await expect(page.getByText("Page 1 of 2")).toBeVisible();
});

test("tournament rows link to a standings detail page", async ({ page }) => {
    await page.goto("/tournaments");

    await page.getByRole("link", { name: /Worlds Warmup Regional/ }).click();

    await expect(page).toHaveURL(/\/tournaments\/tour-01$/);
    await expect(page.getByRole("heading", { name: "Worlds Warmup Regional" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Final Standings" })).toBeVisible();
    await expect(page.getByText("Ash Ketchum")).toBeVisible();
});

test("tournaments page shows empty and server-error fallback states", async ({ page }) => {
    await page.goto("/tournaments?meta_id=meta-empty");
    await expect(page.getByText("No tournaments found")).toBeVisible();

    await page.goto("/tournaments?meta_id=meta-error");
    await expect(page.getByText("No tournaments found")).toBeVisible();
});

test("missing tournaments render the Next.js not-found page", async ({ page }) => {
    await page.goto("/tournaments/missing");

    await expect(page.getByText("This page could not be found.")).toBeVisible();
});
