import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import Card from "./card";

describe("Card", () => {
    it("renders string headings with eyebrow text and an h2", () => {
        render(
            React.createElement(Card, {
                heading: "Overview",
                headingMeta: "Summary",
                children: React.createElement("p", null, "Body content"),
            }),
        );

        expect(screen.getByText("Summary")).toHaveClass("eyebrow");
        expect(
            screen.getByRole("heading", { level: 2, name: "Overview" }),
        ).toBeInTheDocument();
        expect(screen.getByText("Body content")).toBeInTheDocument();
    });

    it("renders ReactNode headings as-is without the generated string wrapper", () => {
        render(
            React.createElement(Card, {
                heading: React.createElement("h3", null, "Custom Heading"),
                headingMeta: React.createElement("span", null, "Meta badge"),
                children: React.createElement("p", null, "Details"),
            }),
        );

        expect(
            screen.getByRole("heading", { level: 3, name: "Custom Heading" }),
        ).toBeInTheDocument();
        expect(screen.queryByRole("heading", { level: 2 })).not.toBeInTheDocument();
        expect(screen.queryByText("Meta badge")?.closest("p")).toBeNull();
        expect(screen.getByText("Meta badge").closest(".muted")).toBeInTheDocument();
        expect(screen.getByText("Details")).toBeInTheDocument();
    });

    it("renders ReactNode heading meta in the side meta slot for string headings", () => {
        render(
            React.createElement(Card, {
                heading: "Overview",
                headingMeta: React.createElement(
                    "span",
                    { "data-testid": "heading-meta" },
                    "Updated now",
                ),
                children: React.createElement("p", null, "Children stay visible"),
            }),
        );

        const meta = screen.getByTestId("heading-meta");
        expect(meta).toBeInTheDocument();
        expect(meta.closest(".muted")).toBeInTheDocument();
        expect(screen.queryByText("Updated now")?.closest("p")).toBeNull();
        expect(screen.getByText("Children stay visible")).toBeInTheDocument();
    });

    it("omits the heading wrapper entirely when no heading is provided", () => {
        const { container } = render(
            React.createElement(
                Card,
                null,
                React.createElement("p", null, "Only content"),
            ),
        );

        expect(container.querySelector(".section__heading")).not.toBeInTheDocument();
        expect(screen.getByText("Only content")).toBeInTheDocument();
    });

    it("merges custom className values and always renders children", () => {
        const { container } = render(
            React.createElement(Card, {
                className: "card--tight extra-spacing",
                children: React.createElement(
                    "button",
                    { type: "button" },
                    "Open",
                ),
            }),
        );

        expect(container.firstElementChild).toHaveClass(
            "card",
            "section",
            "card--tight",
            "extra-spacing",
        );
        expect(screen.getByRole("button", { name: "Open" })).toBeInTheDocument();
    });
});
