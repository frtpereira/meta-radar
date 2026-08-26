import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
    testDir: "./tests",
    timeout: 30000,
    webServer: [
        {
            command: "node ./tests/mock-api-server.mjs",
            cwd: ".",
            url: "http://127.0.0.1:4100/health",
            timeout: 30000,
            reuseExistingServer: true,
        },
        {
            command: "NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:4100/api npm run dev -- --port 3100",
            cwd: ".",
            url: "http://127.0.0.1:3100",
            timeout: 120_000,
            reuseExistingServer: true,
        },
    ],
    use: {
        baseURL: "http://127.0.0.1:3100",
        headless: true,
        trace: "on-first-retry",
    },
    projects: [
        { name: "chromium", use: { ...devices["Desktop Chrome"] } },
        { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    ],
});
