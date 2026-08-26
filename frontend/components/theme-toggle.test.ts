import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { ThemeToggle } from "./theme-toggle";

type MockMediaQueryList = MediaQueryList & {
    dispatch: (matches: boolean) => void;
};

function installMatchMedia(initialMatches: boolean): MockMediaQueryList {
    const listeners = new Set<(event: MediaQueryListEvent) => void>();
    const mediaQueryList = {
        matches: initialMatches,
        media: "(prefers-color-scheme: light)",
        onchange: null,
        addEventListener: (
            _type: string,
            listener: EventListenerOrEventListenerObject,
        ) => {
            listeners.add(listener as (event: MediaQueryListEvent) => void);
        },
        removeEventListener: (
            _type: string,
            listener: EventListenerOrEventListenerObject,
        ) => {
            listeners.delete(listener as (event: MediaQueryListEvent) => void);
        },
        addListener: (listener: (event: MediaQueryListEvent) => void) => {
            listeners.add(listener);
        },
        removeListener: (listener: (event: MediaQueryListEvent) => void) => {
            listeners.delete(listener);
        },
        dispatchEvent: vi.fn(),
        dispatch(matches: boolean) {
            mediaQueryList.matches = matches;
            const event = { matches, media: mediaQueryList.media } as MediaQueryListEvent;
            listeners.forEach((listener) => listener(event));
        },
    } satisfies Partial<MockMediaQueryList>;

    Object.defineProperty(window, "matchMedia", {
        writable: true,
        value: vi.fn().mockImplementation(() => mediaQueryList),
    });

    return mediaQueryList as MockMediaQueryList;
}

describe("ThemeToggle", () => {
    it("reads the initial theme from the data-theme attribute", async () => {
        document.documentElement.setAttribute("data-theme", "light");
        installMatchMedia(false);

        render(React.createElement(ThemeToggle));

        await waitFor(() => {
            expect(
                screen.getByRole("button", { name: "Switch to dark theme" }),
            ).toBeInTheDocument();
        });
    });

    it("falls back to the system preference when no data-theme attribute is set", async () => {
        installMatchMedia(true);

        render(React.createElement(ThemeToggle));

        await waitFor(() => {
            expect(
                screen.getByRole("button", { name: "Switch to dark theme" }),
            ).toBeInTheDocument();
        });
        expect(document.documentElement).not.toHaveAttribute("data-theme");
    });

    it("toggles between dark and light themes and persists the choice", async () => {
        document.documentElement.setAttribute("data-theme", "dark");
        installMatchMedia(false);
        const user = userEvent.setup();

        render(React.createElement(ThemeToggle));

        const button = await screen.findByRole("button", {
            name: "Switch to light theme",
        });
        await user.click(button);

        expect(document.documentElement).toHaveAttribute("data-theme", "light");
        expect(window.localStorage.getItem("theme")).toBe("light");
        expect(
            screen.getByRole("button", { name: "Switch to dark theme" }),
        ).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: "Switch to dark theme" }));
        expect(document.documentElement).toHaveAttribute("data-theme", "dark");
        expect(window.localStorage.getItem("theme")).toBe("dark");
    });

    it("applies live system theme changes when the user has not made an explicit choice", async () => {
        const media = installMatchMedia(false);

        render(React.createElement(ThemeToggle));

        await screen.findByRole("button", { name: "Switch to light theme" });

        act(() => {
            media.dispatch(true);
        });

        expect(document.documentElement).toHaveAttribute("data-theme", "light");
        expect(
            screen.getByRole("button", { name: "Switch to dark theme" }),
        ).toBeInTheDocument();
    });

    it("ignores live system theme changes after an explicit user choice exists", async () => {
        document.documentElement.setAttribute("data-theme", "dark");
        window.localStorage.setItem("theme", "dark");
        const media = installMatchMedia(false);

        render(React.createElement(ThemeToggle));

        await screen.findByRole("button", { name: "Switch to light theme" });

        act(() => {
            media.dispatch(true);
        });

        expect(document.documentElement).toHaveAttribute("data-theme", "dark");
        expect(
            screen.getByRole("button", { name: "Switch to light theme" }),
        ).toBeInTheDocument();
    });
});
