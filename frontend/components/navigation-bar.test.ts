import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { usePathname } from "next/navigation";
import { NavigationBar } from "./navigation-bar";

vi.mock("next/link", () => ({
    default: (props: any) =>
        React.createElement("a", { href: props.href, ...props }, props.children),
}));

vi.mock("next/navigation", () => ({
    usePathname: vi.fn(),
}));

vi.mock("@/components/theme-toggle", () => ({
    ThemeToggle: () =>
        React.createElement("button", { "data-testid": "theme-toggle" }, "Theme"),
}));

const mockedUsePathname = vi.mocked(usePathname);

describe("NavigationBar", () => {
    it("marks only the home link as current on the root path", () => {
        mockedUsePathname.mockReturnValue("/");
        render(React.createElement(NavigationBar));

        expect(screen.getByRole("link", { name: "Home" })).toHaveAttribute(
            "aria-current",
            "page",
        );
        expect(screen.getByRole("link", { name: "Events" })).not.toHaveAttribute(
            "aria-current",
        );
    });

    it.each([
        ["/tournaments", "Events"],
        ["/tournaments/123", "Events"],
        ["/decklists", "Decks"],
        ["/decklists/42", "Decks"],
        ["/matchups", "Matchups"],
        ["/matchups/overview", "Matchups"],
        ["/players", "Players"],
        ["/players/Ash", "Players"],
    ])(
        "marks %s as current for the %s nav item",
        (pathname, currentLabel) => {
            mockedUsePathname.mockReturnValue(pathname);
            render(React.createElement(NavigationBar));

            expect(
                screen.getByRole("link", { name: currentLabel }),
            ).toHaveAttribute("aria-current", "page");
            expect(
                screen.getByRole("link", { name: "Home" }),
            ).not.toHaveAttribute("aria-current");
        },
    );

    it("renders the theme toggle alongside the main navigation", () => {
        mockedUsePathname.mockReturnValue("/decklists");
        render(React.createElement(NavigationBar));

        expect(screen.getByTestId("theme-toggle")).toBeInTheDocument();
        expect(screen.getByRole("navigation", { name: "Main" })).toBeInTheDocument();
    });
});
