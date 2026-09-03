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
    await expect(page.getByText("Lost Box").first()).toBeVisible();
    await expect(page.getByText("Top played deck")).toBeVisible();
    await expect(page.getByText("Latest Doom winner")).toBeVisible();
    await expect(page.getByText("Doom's Local Cup")).toBeVisible();
    await expect(page.getByText("Upcoming set releases")).toBeVisible();
    await expect(page.getByText("30th Anniversary")).toBeVisible();
    await expect(page.getByText("Delta Reign")).toBeVisible();
});

test("homepage handles a meta with no synced data", async ({ page }) => {
    await page.goto("/?meta_id=meta-empty");

    await expect(page.getByText("No tournaments found")).toBeVisible();
    await expect(page.getByText("No archetype stats yet")).toBeVisible();
});

test("homepage tables don't overflow horizontally at common desktop widths", async ({
    page,
}) => {
    await page.setViewportSize({ width: 1024, height: 900 });
    await page.goto("/");

    await expect(page.getByText("Worlds Warmup Regional")).toBeVisible();

    const overflowCounts = await page
        .locator(".table-wrap")
        .evaluateAll((els) =>
            els.filter((el) => el.scrollWidth > el.clientWidth).length,
        );

    expect(overflowCounts).toBe(0);
});
