import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Workaround for Next.js 16 + React 19 global-error prerender bug
  experimental: {
    staticGenerationRetryCount: 0,
  },
};

export default nextConfig;
