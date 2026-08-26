import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import Hero from "./hero";

describe("Hero", () => {
    it("renders title, eyebrow, lede, meta, and actions", () => {
        render(
            React.createElement(Hero, {
                eyebrow: "Eyebrow",
                title: "My Title",
                lede: "Lede text",
                meta: React.createElement("span", null, "Meta content"),
                actions: React.createElement(
                    "button",
                    { type: "button" },
                    "Take action",
                ),
            }),
        );

        expect(screen.getByText("Eyebrow")).toHaveClass("eyebrow");
        expect(
            screen.getByRole("heading", { level: 1, name: "My Title" }),
        ).toBeInTheDocument();
        expect(screen.getByText("Lede text")).toHaveClass("lede");
        expect(screen.getByText("Meta content")).toBeInTheDocument();
        expect(
            screen.getByRole("button", { name: "Take action" }),
        ).toBeInTheDocument();
    });

    it("omits the eyebrow when not provided", () => {
        const { container } = render(React.createElement(Hero, { title: "No Eyebrow" }));

        expect(container.querySelector(".eyebrow")).not.toBeInTheDocument();
        expect(
            screen.getByRole("heading", { level: 1, name: "No Eyebrow" }),
        ).toBeInTheDocument();
    });

    it("omits the lede when not provided", () => {
        const { container } = render(React.createElement(Hero, { title: "No Lede" }));

        expect(container.querySelector(".lede")).not.toBeInTheDocument();
    });

    it("omits meta and actions when they are not provided", () => {
        const { container } = render(React.createElement(Hero, { title: "Minimal Hero" }));

        expect(container.querySelector(".hero__meta")).not.toBeInTheDocument();
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("renders ReactNode titles inside the h1", () => {
        render(
            React.createElement(Hero, {
                title: React.createElement(
                    React.Fragment,
                    null,
                    "Meta ",
                    React.createElement("span", null, "Dashboard"),
                ),
            }),
        );

        const heading = screen.getByRole("heading", { level: 1 });
        expect(heading).toHaveTextContent("Meta Dashboard");
        expect(heading.querySelector("span")).toHaveTextContent("Dashboard");
    });

    it("merges a custom className into the hero container", () => {
        const { container } = render(
            React.createElement(Hero, {
                title: "Styled Hero",
                className: "hero--dense accent-glow",
            }),
        );

        expect(container.firstElementChild).toHaveClass(
            "hero",
            "card",
            "hero--dense",
            "accent-glow",
        );
    });
});
