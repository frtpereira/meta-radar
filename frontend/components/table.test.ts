import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import Table from "./table";

describe("Table", () => {
    it("uses a custom render function when provided and falls back to property access otherwise", () => {
        render(
            React.createElement(Table<{ name: string; wins: number }>, {
                columns: [
                    { key: "name", label: "Name" },
                    {
                        key: "wins",
                        label: "Wins",
                        render: (row) => `${row.wins} wins`,
                    },
                ],
                rows: [{ name: "Gardevoir", wins: 12 }],
            }),
        );

        expect(
            screen.getByRole("columnheader", { name: "Name" }),
        ).toBeInTheDocument();
        expect(
            screen.getByRole("cell", { name: "Gardevoir" }),
        ).toBeInTheDocument();
        expect(
            screen.getByRole("cell", { name: "12 wins" }),
        ).toBeInTheDocument();
    });

    it("renders only the header row when the rows array is empty", () => {
        const { container } = render(
            React.createElement(Table, {
                columns: [
                    { key: "name", label: "Name" },
                    { key: "players", label: "Players" },
                ],
                rows: [],
            }),
        );

        expect(screen.getAllByRole("columnheader")).toHaveLength(2);
        expect(container.querySelectorAll("tbody tr")).toHaveLength(0);
    });

    it("applies per-column class names to multiple headers", () => {
        render(
            React.createElement(Table, {
                columns: [
                    { key: "name", label: "Name", className: "left-col" },
                    {
                        key: "players",
                        label: "Players",
                        className: "right-col",
                    },
                ],
                rows: [{ name: "League Cup", players: 78 }],
            }),
        );

        expect(screen.getByRole("columnheader", { name: "Name" })).toHaveClass(
            "left-col",
        );
        expect(
            screen.getByRole("columnheader", { name: "Players" }),
        ).toHaveClass("right-col");
    });
});
