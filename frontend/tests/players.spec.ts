import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => localStorage.clear());
});

test("players search redirects to the player's history page", async ({ page }) => {
    await page.goto("/players");

    await expect(page.getByRole("heading", { name: "Player Lookup" })).toBeVisible();
    await page
        .getByRole("searchbox", { name: "Player nickname" })
        .fill("Ash Ketchum");
    await page.getByRole("button", { name: "Search" }).click();

    await expect(page).toHaveURL(/\/players\/Ash%20Ketchum$/);
    await expect(page.getByRole("heading", { name: "Ash Ketchum" })).toBeVisible();
});

test("player detail page lists tournament history and links to a decklist", async ({ page }) => {
    await page.goto("/players/Ash Ketchum");

    await expect(page.getByRole("heading", { name: "Ash Ketchum" })).toBeVisible();
    await expect(page.getByText("#1")).toBeVisible();
    await expect(page.getByText("Worlds Warmup Regional")).toBeVisible();
    await expect(page.getByText("Charizard ex")).toBeVisible();
    await expect(page.getByText("Dropped")).toBeVisible();

    await page.getByRole("link", { name: "View decklist" }).click();
    await expect(page).toHaveURL(/\/players\/Ash%20Ketchum\/decklist\/101$/);
    await expect(page.getByRole("heading", { name: "Charizard ex" })).toBeVisible();
    await expect(page.getByText("Charmander")).toBeVisible();
    await expect(page.getByText("Rare Candy")).toBeVisible();
    await expect(page.getByText("Fire Energy")).toBeVisible();
});

test("unknown player nickname shows the not-found page", async ({ page }) => {
    const response = await page.goto("/players/Nobody");
    expect(response?.status()).toBe(404);
});
