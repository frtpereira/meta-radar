import { fileURLToPath } from "node:url";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {
            "@": fileURLToPath(new URL("./", import.meta.url)),
        },
    },
    test: {
        environment: "jsdom",
        globals: true,
        setupFiles: "./vitest.setup.ts",
        include: [
            "components/**/*.test.{ts,tsx}",
            "app/**/*.test.{ts,tsx}",
            "lib/**/*.test.{ts,tsx}",
        ],
        exclude: ["tests/**", "node_modules/**"],
        coverage: {
            provider: "v8",
            reporter: ["text", "lcov"],
        },
    },
});
