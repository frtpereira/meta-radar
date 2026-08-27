"use client";

import React, { useEffect, useMemo, useState } from "react";
import { PaginationControls } from "./pagination";

type SortValue = string | number | null | undefined;

type Column<T> = {
    key: string;
    label: React.ReactNode;
    render?: (row: T) => React.ReactNode;
    className?: string;
    sortable?: boolean;
    sortValue?: (row: T) => SortValue;
    // Some columns (e.g. counts, rates) are more useful high-to-low, so
    // let them start there instead of the usual ascending-first cycle.
    sortDescFirst?: boolean;
};

export type SortState = {
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
    sortState,
    onSortChange,
    pageSize,
}: {
    columns: Column<T>[];
    rows: T[];
    tableClassName?: string;
    // Set to false for short tables (a handful of rows) where sorting
    // adds no value; individual columns can also opt out via
    // `sortable: false` on the Column definition.
    sortable?: boolean;
    // Controlled sort: when both are provided, Table defers sorting to the
    // caller (e.g. a server-side sort driven by URL params) instead of
    // reordering `rows` itself -- `rows` is trusted to already be in the
    // right order.
    sortState?: SortState;
    onSortChange?: (column: { key: string }) => void;
    // When set, Table paginates the (sorted) full `rows` array itself and
    // renders its own controls below the table, instead of expecting the
    // caller to have already sliced `rows` down to one page -- sorting a
    // pre-sliced page can only ever reorder that one page (see components/table.test.ts).
    pageSize?: number;
}) {
    const isControlled = onSortChange !== undefined;
    const [internalSort, setInternalSort] = useState<SortState>(null);
    const sort = isControlled ? (sortState ?? null) : internalSort;

    const sortedRows = useMemo(() => {
        if (isControlled || !sort) {
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
    }, [rows, sort, columns, isControlled]);

    const [page, setPage] = useState(1);
    // A new (or re-filtered) `rows` array means the previous page position
    // no longer means anything -- go back to page 1 rather than risk
    // landing past the end of a shorter result set.
    useEffect(() => {
        setPage(1);
    }, [rows]);

    const totalPages = pageSize
        ? Math.max(1, Math.ceil(sortedRows.length / pageSize))
        : 1;
    const safePage = Math.min(page, totalPages);
    const pagedRows = pageSize
        ? sortedRows.slice((safePage - 1) * pageSize, safePage * pageSize)
        : sortedRows;

    function handleSort(column: Column<T>) {
        if (!sortable || column.sortable === false) {
            return;
        }

        if (onSortChange) {
            onSortChange(column);
            return;
        }

        const firstDirection = column.sortDescFirst ? "desc" : "asc";
        const secondDirection = column.sortDescFirst ? "asc" : "desc";
        setInternalSort((current) => {
            if (!current || current.key !== column.key) {
                return { key: column.key, direction: firstDirection };
            }
            if (current.direction === firstDirection) {
                return { key: column.key, direction: secondDirection };
            }
            return null;
        });
    }

    return (
        <>
            <div className="table-wrap">
                <table className={tableClassName}>
                    <thead>
                        <tr>
                            {columns.map((c) => {
                                const isSortable =
                                    sortable && c.sortable !== false;
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
                                                        ? sort.direction ===
                                                          "asc"
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
                        {pagedRows.map((row, i) => (
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

            {pageSize && totalPages > 1 ? (
                <div
                    style={{
                        display: "flex",
                        flexDirection: "column",
                        gap: "4px",
                        marginTop: "16px",
                    }}
                >
                    <span className="muted">
                        Page {safePage} of {totalPages}
                    </span>
                    <PaginationControls
                        page={safePage}
                        totalPages={totalPages}
                        onPageChange={setPage}
                    />
                </div>
            ) : null}
        </>
    );
}
