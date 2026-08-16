import React from "react";

type Column<T> = {
    key: string;
    label: React.ReactNode;
    render?: (row: T) => React.ReactNode;
    className?: string;
};

export default function Table<T>({
    columns,
    rows,
    tableClassName = "",
}: {
    columns: Column<T>[];
    rows: T[];
    tableClassName?: string;
}) {
    return (
        <div className="table-wrap">
            <table className={tableClassName}>
                <thead>
                    <tr>
                        {columns.map((c) => (
                            <th key={c.key} className={c.className}>
                                {c.label}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {rows.map((row, i) => (
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
