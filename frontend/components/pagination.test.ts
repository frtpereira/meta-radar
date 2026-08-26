import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { beforeEach, describe, expect, it, vi } from "vitest";
import Pagination from "./pagination";

vi.mock("next/navigation", () => ({
    useRouter: vi.fn(),
    usePathname: vi.fn(),
    useSearchParams: vi.fn(),
}));

const mockedUseRouter = vi.mocked(useRouter);
const mockedUsePathname = vi.mocked(usePathname);
const mockedUseSearchParams = vi.mocked(useSearchParams);

describe("Pagination", () => {
    const replace = vi.fn();

    beforeEach(() => {
        replace.mockReset();
        mockedUseRouter.mockReturnValue({ replace } as never);
        mockedUsePathname.mockReturnValue("/tournaments");
        mockedUseSearchParams.mockReturnValue(
            new URLSearchParams("source=online&sort=date") as never,
        );
        vi.mocked(window.scrollTo).mockClear();
    });

    it("disables the previous button on page 1", () => {
        render(React.createElement(Pagination, { page: 1, totalPages: 5 }));

        expect(screen.getByRole("button", { name: "Prev" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
        expect(screen.getByRole("button", { name: "1" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "2" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "3" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "4" })).not.toBeInTheDocument();
    });

    it("disables the next button on the last page", () => {
        render(React.createElement(Pagination, { page: 5, totalPages: 5 }));

        expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Prev" })).toBeEnabled();
        expect(screen.getByRole("button", { name: "3" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "4" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "5" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "2" })).not.toBeInTheDocument();
    });

    it("shows a clamped current-page window in the middle of a large range", () => {
        render(React.createElement(Pagination, { page: 5, totalPages: 10 }));

        ["3", "4", "5", "6", "7"].forEach((name) => {
            expect(screen.getByRole("button", { name })).toBeInTheDocument();
        });
        ["2", "8"].forEach((name) => {
            expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
        });
    });

    it("navigates with router.replace and preserves existing search params", () => {
        render(React.createElement(Pagination, { page: 2, totalPages: 5 }));

        fireEvent.click(screen.getByRole("button", { name: "4" }));

        expect(replace).toHaveBeenCalledWith(
            "/tournaments?source=online&sort=date&page=4",
        );
        expect(window.scrollTo).toHaveBeenCalledWith({
            top: 0,
            behavior: "smooth",
        });
    });

    it("handles a single-page result set", () => {
        render(React.createElement(Pagination, { page: 1, totalPages: 1 }));

        expect(screen.getByRole("button", { name: "Prev" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
        expect(screen.getAllByRole("button")).toHaveLength(3);
        expect(screen.getByRole("button", { name: "1" })).toBeInTheDocument();
    });
});
