import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => localStorage.clear());
});

test("decklists page paginates and filters archetypes in the browser", async ({ page }) => {
    await page.goto("/decklists");

    await expect(page.getByRole("heading", { name: "Deck Archetype Explorer" })).toBeVisible();
    await expect(page.getByText("Page 1 of 2")).toBeVisible();
    await expect(page.getByText("Late Game Dragon")).toHaveCount(0);

    await page.getByRole("button", { name: /^Next$/ }).click();
    await expect(page.getByText("Page 2 of 2")).toBeVisible();
    await expect(page.getByText("Late Game Dragon")).toBeVisible();

    await page.getByRole("searchbox", { name: "Search archetypes" }).fill("late game");
    await expect(page.getByText("1 archetypes")).toBeVisible();
    await expect(page.getByText("Late Game Dragon")).toBeVisible();
    await expect(page.getByText(/Page \d of \d/)).toHaveCount(0);
});

test("decklists page shows empty and server-error fallback states", async ({ page }) => {
    await page.goto("/decklists?meta_id=meta-empty");
    await expect(page.getByText("No archetypes found")).toBeVisible();

    await page.goto("/decklists?meta_id=meta-error");
    await expect(page.getByText("No archetypes found")).toBeVisible();
});

test("decklist detail renders the card list and matchup summaries", async ({ page }) => {
    await page.goto("/decklists/1?meta_id=meta-2026");

    await expect(page.getByRole("heading", { name: "Charizard ex" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Core/Skeleton" })).toBeVisible();
    await expect(page.getByText("Charmander")).toBeVisible();
    await expect(page.getByText("Rare Candy")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Optional Cards" })).toBeVisible();
    await expect(page.getByText("Buddy-Buddy Poffin")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Best Against" })).toBeVisible();
    await expect(page.getByRole("link", { name: "View all matchups →" })).toBeVisible();
});
