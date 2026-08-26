"use client";

import React, { useMemo, useState } from "react";

type SortValue = string | number | null | undefined;

type Column<T> = {
    key: string;
    label: React.ReactNode;
    render?: (row: T) => React.ReactNode;
    className?: string;
    sortable?: boolean;
    sortValue?: (row: T) => SortValue;
};

type SortState = {
    key: string;
    direction: "asc" | "desc";
} | null;

function defaultSortValue<T>(row: T, key: string): SortValue {
    const value = (row as Record<string, unknown>)[key];
    return typeof value === "string" || typeof value === "number"
        ? value
        : null;
}

function compareValues(a: SortValue, b: SortValue) {
    if (a === null || a === undefined) {
        return b === null || b === undefined ? 0 : 1;
    }
    if (b === null || b === undefined) {
        return -1;
    }

    if (typeof a === "number" && typeof b === "number") {
        return a - b;
    }

    return String(a).localeCompare(String(b), undefined, { numeric: true });
}

export default function Table<T>({
    columns,
    rows,
    tableClassName = "",
    sortable = true,
}: {
    columns: Column<T>[];
    rows: T[];
    tableClassName?: string;
    // Set to false for short tables (a handful of rows) where sorting
    // adds no value; individual columns can also opt out via
    // `sortable: false` on the Column definition.
    sortable?: boolean;
}) {
    const [sort, setSort] = useState<SortState>(null);

    const sortedRows = useMemo(() => {
        if (!sort) {
            return rows;
        }

        const column = columns.find((c) => c.key === sort.key);
        if (!column) {
            return rows;
        }

        const getValue =
            column.sortValue ?? ((row: T) => defaultSortValue(row, column.key));
        const direction = sort.direction === "asc" ? 1 : -1;

        return [...rows].sort(
            (a, b) => compareValues(getValue(a), getValue(b)) * direction,
        );
    }, [rows, sort, columns]);

    function handleSort(column: Column<T>) {
        if (!sortable || column.sortable === false) {
            return;
        }

        setSort((current) => {
            if (!current || current.key !== column.key) {
                return { key: column.key, direction: "asc" };
            }
            if (current.direction === "asc") {
                return { key: column.key, direction: "desc" };
            }
            return null;
        });
    }

    return (
        <div className="table-wrap">
            <table className={tableClassName}>
                <thead>
                    <tr>
                        {columns.map((c) => {
                            const isSortable = sortable && c.sortable !== false;
                            const isActive = sort?.key === c.key;
                            const ariaSort = isSortable
                                ? isActive
                                    ? sort.direction === "asc"
                                        ? "ascending"
                                        : "descending"
                                    : "none"
                                : undefined;

                            return (
                                <th
                                    key={c.key}
                                    className={`${c.className ?? ""} ${isSortable ? "th-sortable" : ""}`.trim()}
                                    aria-sort={ariaSort}
                                >
                                    {isSortable ? (
                                        <button
                                            type="button"
                                            className="th-sort-button"
                                            onClick={() => handleSort(c)}
                                        >
                                            {c.label}
                                            <span
                                                className="th-sort-indicator"
                                                aria-hidden="true"
                                            >
                                                {isActive
                                                    ? sort.direction === "asc"
                                                        ? "▲"
                                                        : "▼"
                                                    : "↕"}
                                            </span>
                                        </button>
                                    ) : (
                                        c.label
                                    )}
                                </th>
                            );
                        })}
                    </tr>
                </thead>
                <tbody>
                    {sortedRows.map((row, i) => (
                        <tr key={i}>
                            {columns.map((c) => (
                                <td key={c.key}>
                                    {c.render
                                        ? c.render(row)
                                        : (row as any)[c.key]}
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
