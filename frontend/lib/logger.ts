/**
 * Paths that should not be logged
 */
const SILENT_PATHS = ["/card-images"];

/**
 * ANSI color codes for terminal output
 */
const COLORS = {
    reset: "\x1b[0m",
    green: "\x1b[32m",
    yellow: "\x1b[33m",
    red: "\x1b[31m",
    gray: "\x1b[90m",
};

/**
 * Get color for status code
 * Green: 2xx, Yellow: 3xx, Red: 4xx/5xx
 */
function getStatusColor(status: number): string {
    if (status >= 200 && status < 300) return COLORS.green;
    if (status >= 300 && status < 400) return COLORS.yellow;
    return COLORS.red;
}

/**
 * Formats current time as [HH:mm:ss.SSS]
 */
function formatTimestamp(): string {
    const now = new Date();
    const hours = String(now.getHours()).padStart(2, "0");
    const minutes = String(now.getMinutes()).padStart(2, "0");
    const seconds = String(now.getSeconds()).padStart(2, "0");
    const millis = String(now.getMilliseconds()).padStart(3, "0");
    return `[${hours}:${minutes}:${seconds}.${millis}]`;
}

/**
 * Check if a path should be logged
 */
function shouldLog(path: string): boolean {
    return !SILENT_PATHS.some((silentPath) => path.startsWith(silentPath));
}

/**
 * Logs HTTP requests in the same format as Next.js with timestamp and colored status
 * Format: METHOD PATH STATUS in XXXms [HH:mm:ss.SSS]
 * Status codes are colored: green (2xx), yellow (3xx), red (4xx/5xx)
 */
export function logRequest(
    method: string,
    path: string,
    status: number,
    durationMs: number,
) {
    if (!shouldLog(path)) {
        return;
    }
    const timestamp = formatTimestamp();
    const statusColor = getStatusColor(status);
    const coloredStatus = `${statusColor}${status}${COLORS.reset}`;
    console.log(
        `${method} ${path} ${coloredStatus} in ${durationMs}ms ${COLORS.gray}${timestamp}${COLORS.reset}`,
    );
}

/**
 * Helper to measure and log fetch requests
 */
export async function fetchWithLogging<T>(
    url: string,
    options?: RequestInit,
): Promise<{ response: Response; text: string }> {
    const startTime = performance.now();
    const method = options?.method ?? "GET";

    // Extract path from URL for logging (remove domain)
    const urlObj = new URL(url, "http://localhost");
    const path = urlObj.pathname + urlObj.search;

    try {
        const response = await fetch(url, options);
        const text = await response.text();
        const durationMs = Math.round(performance.now() - startTime);

        logRequest(method, path, response.status, durationMs);

        return { response, text };
    } catch (error) {
        const durationMs = Math.round(performance.now() - startTime);
        logRequest(method, path, 0, durationMs); // 0 for network errors
        throw error;
    }
}
