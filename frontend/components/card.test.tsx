import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import Card from "./card";

describe("Card", () => {
    it("renders string headings with eyebrow text and an h2", () => {
        render(
            <Card heading="Overview" headingMeta="Summary">
                <p>Body content</p>
            </Card>,
        );

        expect(screen.getByText("Summary")).toHaveClass("eyebrow");
        expect(
            screen.getByRole("heading", { level: 2, name: "Overview" }),
        ).toBeInTheDocument();
        expect(screen.getByText("Body content")).toBeInTheDocument();
    });

    it("renders ReactNode headings as-is without the generated string wrapper", () => {
        render(
            <Card
                heading={<h3>Custom Heading</h3>}
                headingMeta={<span>Meta badge</span>}
            >
                <p>Details</p>
            </Card>,
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
            <Card
                heading="Overview"
                headingMeta={<span data-testid="heading-meta">Updated now</span>}
            >
                <p>Children stay visible</p>
            </Card>,
        );

        const meta = screen.getByTestId("heading-meta");
        expect(meta).toBeInTheDocument();
        expect(meta.closest(".muted")).toBeInTheDocument();
        expect(screen.queryByText("Updated now")?.closest("p")).toBeNull();
        expect(screen.getByText("Children stay visible")).toBeInTheDocument();
    });

    it("omits the heading wrapper entirely when no heading is provided", () => {
        const { container } = render(
            <Card>
                <p>Only content</p>
            </Card>,
        );

        expect(container.querySelector(".section__heading")).not.toBeInTheDocument();
        expect(screen.getByText("Only content")).toBeInTheDocument();
    });

    it("merges custom className values and always renders children", () => {
        const { container } = render(
            <Card className="card--tight extra-spacing">
                <button type="button">Open</button>
            </Card>,
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
