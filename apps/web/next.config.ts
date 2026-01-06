import type { NextConfig } from "next";

const isDev = process.env.NODE_ENV === 'development';

// API URL for CSP (allow connections to the backend)
const apiUrl = process.env.NEXT_PUBLIC_API_URL || '';

// In development, allow localhost connections for local API
// In production, allow the configured API URL
const connectSrc = isDev
  ? "'self' http://127.0.0.1:* http://localhost:* ws://127.0.0.1:* ws://localhost:* https://www.google-analytics.com"
  : `'self' ${apiUrl} https://www.google-analytics.com`;

const securityHeaders = [
  {
    key: 'X-DNS-Prefetch-Control',
    value: 'on',
  },
  {
    key: 'Strict-Transport-Security',
    value: 'max-age=31536000; includeSubDomains; preload',
  },
  {
    key: 'X-Frame-Options',
    value: 'DENY',
  },
  {
    key: 'X-Content-Type-Options',
    value: 'nosniff',
  },
  {
    key: 'Referrer-Policy',
    value: 'strict-origin-when-cross-origin',
  },
  {
    key: 'X-XSS-Protection',
    value: '1; mode=block',
  },
  {
    key: 'Permissions-Policy',
    value: 'camera=(), microphone=(), geolocation=()',
  },
  {
    key: 'Content-Security-Policy',
    value: [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://www.googletagmanager.com https://www.google-analytics.com",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: https: blob:",
      "font-src 'self' data:",
      `connect-src ${connectSrc}`,
      "frame-ancestors 'none'",
      "base-uri 'self'",
      "form-action 'self'",
    ].join('; '),
  },
];

const nextConfig: NextConfig = {
  output: 'standalone',
  reactCompiler: true,
  // Workaround for Next.js 16 + React 19 global-error prerender bug
  // See: https://github.com/vercel/next.js/issues/85668
  experimental: {
    staticGenerationRetryCount: 0,
  },
  async headers() {
    return [
      {
        source: '/:path*',
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;
