import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import InfoTooltip from "./info-tooltip";

describe("InfoTooltip", () => {
    it("renders the native title attribute with the supplied text", () => {
        render(React.createElement(InfoTooltip, { text: "Explains score rate" }));

        expect(screen.getByTitle("Explains score rate")).toBeInTheDocument();
    });

    it("renders visually hidden text for screen readers", () => {
        const { container } = render(
            React.createElement(InfoTooltip, { text: "Accessible help copy" }),
        );

        const srOnly = container.querySelector(".sr-only");
        expect(srOnly).toHaveTextContent("Accessible help copy");
    });

    it("is keyboard focusable via tabIndex 0", () => {
        const { container } = render(
            React.createElement(InfoTooltip, { text: "Focusable tooltip" }),
        );

        expect(container.querySelector(".info-tooltip")).toHaveAttribute(
            "tabindex",
            "0",
        );
    });
});
