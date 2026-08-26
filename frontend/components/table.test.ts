import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it } from "vitest";
import Table from "./table";

function firstColumnValues() {
    return screen
        .getAllByRole("row")
        .slice(1)
        .map((row) => within(row).getAllByRole("cell")[0].textContent);
}

function sortButton(name: string | RegExp) {
    return within(screen.getByRole("columnheader", { name })).getByRole(
        "button",
    );
}

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

    describe("sorting", () => {
        const rows = [
            { name: "Gardevoir ex", wins: 12 },
            { name: "Charizard ex", wins: 20 },
            { name: "Lugia VSTAR", wins: 5 },
        ];

        it("is unsorted, then ascending, then descending, then unsorted again on repeated clicks", async () => {
            const user = userEvent.setup();
            render(
                React.createElement(Table<{ name: string; wins: number }>, {
                    columns: [
                        { key: "name", label: "Name" },
                        { key: "wins", label: "Wins" },
                    ],
                    rows,
                }),
            );

            expect(firstColumnValues()).toEqual([
                "Gardevoir ex",
                "Charizard ex",
                "Lugia VSTAR",
            ]);

            const nameHeader = screen.getByRole("columnheader", {
                name: /Name/,
            });
            const nameSortButton = within(nameHeader).getByRole("button");

            await user.click(nameSortButton);
            expect(nameHeader).toHaveAttribute("aria-sort", "ascending");
            expect(firstColumnValues()).toEqual([
                "Charizard ex",
                "Gardevoir ex",
                "Lugia VSTAR",
            ]);

            await user.click(nameSortButton);
            expect(nameHeader).toHaveAttribute("aria-sort", "descending");
            expect(firstColumnValues()).toEqual([
                "Lugia VSTAR",
                "Gardevoir ex",
                "Charizard ex",
            ]);

            await user.click(nameSortButton);
            expect(nameHeader).toHaveAttribute("aria-sort", "none");
            expect(firstColumnValues()).toEqual([
                "Gardevoir ex",
                "Charizard ex",
                "Lugia VSTAR",
            ]);
        });

        it("sorts numeric columns numerically rather than lexicographically", async () => {
            const user = userEvent.setup();
            render(
                React.createElement(Table<{ name: string; wins: number }>, {
                    columns: [
                        { key: "name", label: "Name" },
                        { key: "wins", label: "Wins" },
                    ],
                    rows,
                }),
            );

            await user.click(sortButton(/Wins/));

            const winsCells = screen
                .getAllByRole("row")
                .slice(1)
                .map((row) => within(row).getAllByRole("cell")[1].textContent);
            expect(winsCells).toEqual(["5", "12", "20"]);
        });

        it("uses a column's sortValue instead of its rendered/raw value when provided", async () => {
            const user = userEvent.setup();
            render(
                React.createElement(
                    Table<{ name: string; wins: number; losses: number }>,
                    {
                        columns: [
                            { key: "name", label: "Name" },
                            {
                                key: "record",
                                label: "Record",
                                render: (row) => `${row.wins}-${row.losses}`,
                                sortValue: (row) => row.wins - row.losses,
                            },
                        ],
                        rows: [
                            { name: "A", wins: 3, losses: 10 },
                            { name: "B", wins: 8, losses: 1 },
                            { name: "C", wins: 5, losses: 5 },
                        ],
                    },
                ),
            );

            await user.click(sortButton(/Record/));

            expect(firstColumnValues()).toEqual(["A", "C", "B"]);
        });

        it("keeps null/undefined sortValue results grouped at the end of the direction being applied", async () => {
            const user = userEvent.setup();
            render(
                React.createElement(
                    Table<{ name: string; score: number | null }>,
                    {
                        columns: [
                            { key: "name", label: "Name" },
                            {
                                key: "score",
                                label: "Score",
                                sortValue: (row) => row.score,
                            },
                        ],
                        rows: [
                            { name: "A", score: 2 },
                            { name: "B", score: null },
                            { name: "C", score: 1 },
                        ],
                    },
                ),
            );

            const scoreSortButton = sortButton(/Score/);

            // Ascending: nulls sort after all comparable values.
            await user.click(scoreSortButton);
            expect(firstColumnValues()).toEqual(["C", "A", "B"]);

            // Descending: reversing the ascending comparison also flips
            // where the nulls land, so they end up first.
            await user.click(scoreSortButton);
            expect(firstColumnValues()).toEqual(["B", "A", "C"]);
        });

        it("does not render a sort button and ignores clicks for a column marked sortable: false", async () => {
            const user = userEvent.setup();
            render(
                React.createElement(Table<{ name: string; wins: number }>, {
                    columns: [
                        { key: "name", label: "Name", sortable: false },
                        { key: "wins", label: "Wins" },
                    ],
                    rows,
                }),
            );

            const nameHeader = screen.getByRole("columnheader", {
                name: "Name",
            });
            expect(
                within(nameHeader).queryByRole("button"),
            ).not.toBeInTheDocument();
            expect(nameHeader).not.toHaveAttribute("aria-sort");

            await user.click(nameHeader);
            expect(firstColumnValues()).toEqual([
                "Gardevoir ex",
                "Charizard ex",
                "Lugia VSTAR",
            ]);
        });

        it("disables sorting for every column when the table is sortable={false}", () => {
            render(
                React.createElement(Table<{ name: string; wins: number }>, {
                    columns: [
                        { key: "name", label: "Name" },
                        { key: "wins", label: "Wins" },
                    ],
                    rows,
                    sortable: false,
                }),
            );

            for (const header of screen.getAllByRole("columnheader")) {
                expect(within(header).queryByRole("button")).toBeNull();
                expect(header).not.toHaveAttribute("aria-sort");
            }
        });
    });
});
