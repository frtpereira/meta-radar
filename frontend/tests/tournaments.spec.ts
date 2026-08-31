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

test("sorts client-side, without a page reload, when a filtered result fits on one page", async ({ page }) => {
    await page.goto("/tournaments?min_players=100");

    await expect(page.getByText("Page 1 of 1")).toBeVisible();

    const eventHeader = page.getByRole("columnheader", { name: "Event" });
    await expect(eventHeader.getByRole("button")).toBeVisible();

    function firstEventName() {
        return page
            .locator("tbody tr")
            .first()
            .locator(".table-title")
            .innerText();
    }

    expect(await firstEventName()).toBe("Tournament 18");

    await eventHeader.getByRole("button").click();
    await expect(page).not.toHaveURL(/sort_by/);
    expect(await firstEventName()).toBe("Tournament 18");

    await eventHeader.getByRole("button").click();
    await expect(page).not.toHaveURL(/sort_by/);
    expect(await firstEventName()).toBe("Tournament 25");
});

test("tournament rows link to a standings detail page", async ({ page }) => {
    await page.goto("/tournaments");

    await page.getByRole("link", { name: /Worlds Warmup Regional/ }).click();

    await expect(page).toHaveURL(/\/tournaments\/tour-01$/);
    await expect(page.getByRole("heading", { name: "Worlds Warmup Regional" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Final Standings" })).toBeVisible();
    await expect(page.getByText("Ash Ketchum")).toBeVisible();
});

test("standings rows link to a player's profile page", async ({ page }) => {
    await page.goto("/tournaments/tour-01");

    await page.getByRole("link", { name: "Ash Ketchum" }).click();

    await expect(page).toHaveURL(/\/players\/Ash%20Ketchum$/);
    await expect(page.getByRole("heading", { name: "Ash Ketchum" })).toBeVisible();
});

test("standings rows link to the player's decklist", async ({ page }) => {
    await page.goto("/tournaments/tour-01");

    await page
        .getByRole("row", { name: /Ash Ketchum/ })
        .getByRole("link", { name: "View decklist" })
        .click();

    await expect(page).toHaveURL(/\/players\/Ash%20Ketchum\/decklist\/101$/);
    await expect(page.getByRole("heading", { name: "Charizard ex" })).toBeVisible();
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
