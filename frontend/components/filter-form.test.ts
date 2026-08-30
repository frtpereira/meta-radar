import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { usePathname, useRouter } from "next/navigation";
import { beforeEach, describe, expect, it, vi } from "vitest";
import FilterForm from "./filter-form";

vi.mock("next/navigation", () => ({
    useRouter: vi.fn(),
    usePathname: vi.fn(),
}));

const mockedUseRouter = vi.mocked(useRouter);
const mockedUsePathname = vi.mocked(usePathname);

describe("FilterForm", () => {
    const push = vi.fn();

    beforeEach(() => {
        push.mockReset();
        mockedUseRouter.mockReturnValue({ push } as never);
        mockedUsePathname.mockReturnValue("/tournaments");
    });

    it("omits blank text/select fields and zero-valued number fields from the URL", () => {
        render(
            React.createElement(
                FilterForm,
                null,
                React.createElement("input", {
                    name: "event_name",
                    defaultValue: "",
                }),
                React.createElement(
                    "select",
                    { name: "source", defaultValue: "" },
                    React.createElement("option", { value: "" }, "All sources"),
                    React.createElement("option", { value: "online" }, "Online"),
                ),
                React.createElement("input", {
                    name: "min_players",
                    type: "number",
                    defaultValue: "0",
                }),
                React.createElement("input", {
                    name: "meta_id",
                    defaultValue: "meta-1",
                }),
                React.createElement("button", { type: "submit" }, "Apply"),
            ),
        );

        fireEvent.click(screen.getByRole("button", { name: "Apply" }));

        expect(push).toHaveBeenCalledWith("/tournaments?meta_id=meta-1");
    });

    it("keeps non-empty and non-zero values in the URL", () => {
        render(
            React.createElement(
                FilterForm,
                null,
                React.createElement("input", {
                    name: "event_name",
                    defaultValue: "Regional",
                }),
                React.createElement("input", {
                    name: "min_players",
                    type: "number",
                    defaultValue: "8",
                }),
                React.createElement("button", { type: "submit" }, "Apply"),
            ),
        );

        fireEvent.click(screen.getByRole("button", { name: "Apply" }));

        expect(push).toHaveBeenCalledWith(
            "/tournaments?event_name=Regional&min_players=8",
        );
    });

    it("navigates to the bare pathname when every field is empty", () => {
        render(
            React.createElement(
                FilterForm,
                null,
                React.createElement("input", {
                    name: "event_name",
                    defaultValue: "",
                }),
                React.createElement("button", { type: "submit" }, "Apply"),
            ),
        );

        fireEvent.click(screen.getByRole("button", { name: "Apply" }));

        expect(push).toHaveBeenCalledWith("/tournaments");
    });
});
