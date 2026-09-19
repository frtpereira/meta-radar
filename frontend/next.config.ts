import type { NextConfig } from "next";

const nextConfig: NextConfig = {
    reactStrictMode: true,
    logging: false,
    allowedDevOrigins: [
        "local-origin.dev",
        "*.local-origin.dev",
        "metaradar-tcg.com",
        "*.metaradar-tcg.com",
    ],
};

export default nextConfig;
