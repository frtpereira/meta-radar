import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import CardHoverPreview from "./card-hover-preview";

const PREVIEW_WIDTH = 220;
const PREVIEW_HEIGHT = (PREVIEW_WIDTH * 7) / 5;
const OFFSET = 20;

function setViewport(width: number, height: number) {
    Object.defineProperty(window, "innerWidth", {
        writable: true,
        configurable: true,
        value: width,
    });
    Object.defineProperty(window, "innerHeight", {
        writable: true,
        configurable: true,
        value: height,
    });
}

describe("CardHoverPreview", () => {
    it("renders children unchanged when no imageUrl is available", () => {
        render(
            React.createElement(CardHoverPreview, {
                name: "Test Card",
                children: "Test Card",
            }),
        );

        expect(screen.getByText("Test Card")).toBeInTheDocument();
    });

    it("positions the preview below and to the right of the cursor when there is room", async () => {
        setViewport(1200, 1200);
        render(
            React.createElement(CardHoverPreview, {
                imageUrl: "/card.png",
                name: "Test Card",
                children: "Test Card",
            }),
        );

        fireEvent.mouseEnter(screen.getByText("Test Card"), {
            clientX: 100,
            clientY: 100,
        });

        const img = screen.getByAltText("Test Card");
        await waitFor(() => expect(img.style.left).toBe(`${100 + OFFSET}px`));
        expect(img.style.top).toBe(`${100 + OFFSET}px`);
    });

    it("flips above the cursor when the preview would overflow the bottom of the viewport", async () => {
        setViewport(1200, 400);
        render(
            React.createElement(CardHoverPreview, {
                imageUrl: "/card.png",
                name: "Test Card",
                children: "Test Card",
            }),
        );

        fireEvent.mouseEnter(screen.getByText("Test Card"), {
            clientX: 100,
            clientY: 350,
        });

        const img = screen.getByAltText("Test Card");
        await waitFor(() =>
            expect(img.style.top).toBe(`${350 - OFFSET - PREVIEW_HEIGHT}px`),
        );
    });

    it("flips to the left of the cursor when the preview would overflow the right of the viewport", async () => {
        setViewport(500, 1200);
        render(
            React.createElement(CardHoverPreview, {
                imageUrl: "/card.png",
                name: "Test Card",
                children: "Test Card",
            }),
        );

        fireEvent.mouseEnter(screen.getByText("Test Card"), {
            clientX: 450,
            clientY: 100,
        });

        const img = screen.getByAltText("Test Card");
        await waitFor(() =>
            expect(img.style.left).toBe(`${450 - OFFSET - PREVIEW_WIDTH}px`),
        );
    });
});
