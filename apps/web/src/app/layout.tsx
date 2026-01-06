import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import Script from "next/script";
import { ThemeProvider } from "@/components/theme-provider";
import { AuthProvider } from "@/lib/auth-context";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const siteUrl = "https://lelemon.dev";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Lelemon - Lightweight LLM Observability",
    template: "%s | Lelemon",
  },
  description:
    "Trace your AI agents with zero overhead. Lelemon is a lightweight observability platform for LLMs — track prompts, decisions, and metrics in real-time. <2KB SDK, zero config.",
  keywords: [
    "LLM observability",
    "AI monitoring",
    "OpenAI tracing",
    "Anthropic monitoring",
    "AI agent debugging",
    "LLM analytics",
    "prompt tracing",
    "AI observability",
    "lightweight observability",
  ],
  authors: [{ name: "Lelemon", url: siteUrl }],
  creator: "Lelemon",
  publisher: "Lelemon",
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  manifest: "/site.webmanifest",
  openGraph: {
    type: "website",
    locale: "en_US",
    url: siteUrl,
    siteName: "Lelemon",
    title: "Lelemon - Lightweight LLM Observability",
    description:
      "Trace your AI agents with zero overhead. Track prompts, decisions, and metrics in real-time. <2KB SDK, zero config.",
  },
  twitter: {
    card: "summary_large_image",
    title: "Lelemon - Lightweight LLM Observability",
    description:
      "Trace your AI agents with zero overhead. <2KB SDK, zero config.",
    creator: "@lelemondev",
  },
  alternates: {
    canonical: siteUrl,
  },
  category: "technology",
};

const isProduction = process.env.NODE_ENV === "production";

const jsonLd = {
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  name: "Lelemon",
  applicationCategory: "DeveloperApplication",
  operatingSystem: "Cross-platform",
  description:
    "Lightweight LLM observability platform for AI agents. Trace prompts, decisions, and metrics in real-time with zero overhead.",
  url: "https://lelemon.dev",
  offers: {
    "@type": "Offer",
    price: "0",
    priceCurrency: "USD",
    description: "Free tier available",
  },
  featureList: [
    "LLM call tracing",
    "Token usage tracking",
    "Cost analytics",
    "Multi-provider support (OpenAI, Anthropic, Bedrock, Gemini)",
    "Real-time dashboard",
    "Zero configuration SDK",
  ],
  author: {
    "@type": "Organization",
    name: "Lelemon",
    url: "https://lelemon.dev",
  },
  aggregateRating: {
    "@type": "AggregateRating",
    ratingValue: "5",
    ratingCount: "1",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
        {isProduction && (
          <>
            <Script
              src="https://www.googletagmanager.com/gtag/js?id=G-549BVLXXBW"
              strategy="afterInteractive"
            />
            <Script id="google-analytics" strategy="afterInteractive">
              {`
                window.dataLayer = window.dataLayer || [];
                function gtag(){dataLayer.push(arguments);}
                gtag('js', new Date());
                gtag('config', 'G-549BVLXXBW');
              `}
            </Script>
          </>
        )}
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <AuthProvider>
        <ThemeProvider
          attribute="class"
          defaultTheme="dark"
          enableSystem
          disableTransitionOnChange
        >
          {children}
        </ThemeProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
