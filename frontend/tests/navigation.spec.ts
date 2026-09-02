import { expect, test } from "@playwright/test";

test("main navigation moves between pages and marks the current route", async ({
    page,
}) => {
    await page.goto("/");

    const nav = page.getByRole("navigation", { name: "Main" });
    await expect(nav.getByRole("link", { name: "Home" })).toHaveAttribute(
        "aria-current",
        "page",
    );

    await nav.getByRole("link", { name: "Events" }).click();
    await expect(page).toHaveURL(/\/tournaments$/);
    await expect(nav.getByRole("link", { name: "Events" })).toHaveAttribute(
        "aria-current",
        "page",
    );

    await nav.getByRole("link", { name: "Archetypes" }).click();
    await expect(page).toHaveURL(/\/archetypes$/);
    await expect(nav.getByRole("link", { name: "Archetypes" })).toHaveAttribute(
        "aria-current",
        "page",
    );

    await nav.getByRole("link", { name: "Matchups" }).click();
    await expect(page).toHaveURL(/\/matchups$/);
    await expect(nav.getByRole("link", { name: "Matchups" })).toHaveAttribute(
        "aria-current",
        "page",
    );
});

test("theme toggle persists the chosen theme across reloads", async ({
    page,
}) => {
    await page.goto("/");

    const root = page.locator("html");
    await expect(root).toHaveAttribute("data-theme", /light|dark/);
    const initialTheme = await root.getAttribute("data-theme");
    const nextTheme = initialTheme === "light" ? "dark" : "light";

    await page.locator(".theme-toggle").click();
    await expect(root).toHaveAttribute("data-theme", nextTheme);
    await expect
        .poll(() => page.evaluate(() => localStorage.getItem("theme")))
        .toBe(nextTheme);

    await page.reload();
    await expect(root).toHaveAttribute("data-theme", nextTheme);
});
