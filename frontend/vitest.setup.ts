import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    document.documentElement.removeAttribute("data-theme");
    window.localStorage.clear();
});

Object.defineProperty(window, "scrollTo", {
    writable: true,
    value: vi.fn(),
});
