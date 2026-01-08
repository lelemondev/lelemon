'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/cjs/styles/prism';
import { LemonIcon } from '@/components/lemon-icon';


function CodeBlock() {
  const [copied, setCopied] = useState(false);

  const code = `import { init, observe } from '@lelemondev/sdk';
import OpenAI from 'openai';

init({ apiKey: process.env.LELEMON_API_KEY });

const openai = observe(new OpenAI());

const res = await openai.chat.completions.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: 'Hello!' }],
});`;

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      {/* Glow effect - subtle */}
      <div className="absolute -inset-1 bg-gradient-to-r from-[#FACC15]/15 via-[#FEF08A]/8 to-[#FACC15]/15 rounded-2xl blur-xl opacity-0 group-hover:opacity-100 transition-opacity duration-500" />

      <div className="relative rounded-2xl bg-[#0F0F10] overflow-hidden shadow-xl border border-white/5">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/10 bg-gradient-to-r from-[#1A1A1B] to-[#0F0F10]">
          <div className="flex items-center gap-3">
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-[#FF5F57]" />
              <div className="w-3 h-3 rounded-full bg-[#FFBD2E]" />
              <div className="w-3 h-3 rounded-full bg-[#28C840]" />
            </div>
            <span className="text-sm text-white/30 font-mono">app.ts</span>
          </div>
          <button
            onClick={handleCopy}
            className="flex items-center gap-2 text-xs text-white/40 hover:text-white/80 transition-colors px-3 py-1.5 rounded-lg hover:bg-white/5"
          >
            {copied ? (
              <>
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Copied!
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                Copy
              </>
            )}
          </button>
        </div>

        {/* Code */}
        <div className="p-4 sm:p-6 text-[11px] sm:text-[13px] font-mono leading-6 sm:leading-7 overflow-x-auto">
          <SyntaxHighlighter
            language="typescript"
            style={vscDarkPlus}
            customStyle={{
              backgroundColor: 'transparent',
              fontSize: 'inherit',
              lineHeight: 'inherit',
            }}
            codeTagProps={{
              style: {
                fontFamily: 'inherit',
              }
            }}
          >
            {code}
          </SyntaxHighlighter>
        </div>
      </div>
    </div>
  );
}

function FeatureCheck({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-3 group">
      <div className="mt-1 w-5 h-5 rounded-full bg-gradient-to-br from-[#FEF08A] to-[#FACC15] flex items-center justify-center shadow-[0_2px_6px_-2px_rgba(250,204,21,0.4)] group-hover:shadow-[0_2px_8px_-2px_rgba(250,204,21,0.5)] transition-shadow">
        <svg className="w-3 h-3 text-[#18181B]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
        </svg>
      </div>
      <span className="text-[#3F3F46] leading-relaxed">{children}</span>
    </li>
  );
}

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-[#FAFDF7] text-[#18181B] overflow-hidden">
      {/* Background elements */}
      <div className="fixed inset-0 pointer-events-none">
        {/* Subtle grid */}
        <div className="absolute inset-0 opacity-[0.02]" style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M0 0h60v60H0z' fill='none' stroke='%2318181B' stroke-width='0.5'/%3E%3C/svg%3E")`,
        }} />
        {/* Gradient blobs - subtle */}
        <div className="absolute top-0 right-0 w-[600px] h-[600px] bg-gradient-to-bl from-[#FACC15]/15 via-[#FEF08A]/8 to-transparent rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 w-[400px] h-[400px] bg-gradient-to-tr from-[#FACC15]/8 to-transparent rounded-full blur-3xl" />
      </div>

      {/* Content */}
      <div className="relative">
        {/* Header */}
        <header className="fixed top-0 left-0 right-0 z-50 bg-[#FAFDF7]/80 backdrop-blur-md border-b border-[#18181B]/5">
          <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
            <Link href="/" className="flex items-center gap-2.5 group">
              <div className="relative">
                <LemonIcon className="w-9 h-9 transition-transform group-hover:rotate-12 group-hover:scale-110" />
              </div>
              <span className="font-bold text-xl tracking-tight">Lelemon</span>
            </Link>
          </div>
        </header>

        {/* Hero Section */}
        <section className="pt-32 pb-20 px-6">
          <div className="max-w-4xl mx-auto text-center">
            {/* Badges */}
            <div className="flex flex-wrap items-center justify-center gap-3 mb-8">
              <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[#FACC15]/10 border border-[#FACC15]/20">
                <span className="text-sm">🍋</span>
                <span className="text-sm font-medium text-[#A16207]">LLM Observability</span>
              </div>
              <a
                href="https://github.com/lelemondev/lelemon"
                target="_blank"
                rel="noopener noreferrer"
                className="relative inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-white/80 backdrop-blur-sm border border-[#18181B]/10 hover:border-[#18181B]/20 hover:bg-white/95 text-[#18181B] shadow-[0_2px_8px_-2px_rgba(24,24,27,0.08)] hover:shadow-[0_4px_12px_-4px_rgba(24,24,27,0.12)] hover:-translate-y-0.5 transition-all duration-300 group overflow-hidden"
              >
                {/* Shine effect */}
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-[#FACC15]/5 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700" />
                
                <svg className="w-4 h-4 text-[#18181B] relative z-10 group-hover:scale-110 transition-transform duration-300" fill="currentColor" viewBox="0 0 24 24">
                  <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                </svg>
                <span className="text-sm font-semibold text-[#18181B] relative z-10 group-hover:tracking-wide transition-all duration-300">Open Source</span>
              </a>
            </div>

            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight leading-[1.1] mb-6">
              Fresh, Open Source
              <br />
              <span className="relative inline-block">
                LLM Observability
                <svg className="absolute -bottom-2 left-0 w-full h-3 text-[#FACC15]" viewBox="0 0 200 12" fill="none">
                  <path d="M2 10C50 4 150 4 198 10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
                </svg>
              </span>
              {' '}
              <span className="inline-block animate-bounce-slow">🍋</span>
            </h1>

            <p className="text-xl text-[#3F3F46] leading-relaxed max-w-2xl mx-auto mb-8">
              Stop guessing what your agents are doing. Trace execution flows, debug tool calls, and control costs with zero latency overhead. Squeeze the best out of your stack.
            </p>

            {/* Lightness badges */}
            <div className="flex flex-wrap items-center justify-center gap-4 sm:gap-6 mb-10 text-sm">
              <span className="flex items-center gap-1.5 text-[#3F3F46]">
                <svg className="w-4 h-4 text-[#FACC15]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                Zero config
              </span>
              <span className="flex items-center gap-1.5 text-[#3F3F46]">
                <svg className="w-4 h-4 text-[#22C55E]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                </svg>
                Super lightweight SDK
              </span>
              <span className="flex items-center gap-1.5 text-[#3F3F46]">
                <svg className="w-4 h-4 text-[#FACC15]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                No overhead
              </span>
            </div>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-10">
              <a
                href="#demo"
                className="group inline-flex items-center gap-2 px-7 py-4 bg-[#FACC15] text-[#18181B] font-semibold rounded-full hover:bg-[#EAB308] transition-all shadow-[0_2px_12px_-4px_rgba(250,204,21,0.4)] hover:shadow-[0_4px_16px_-4px_rgba(250,204,21,0.5)] hover:-translate-y-0.5"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.91 11.672a.375.375 0 010 .656l-5.603 3.113a.375.375 0 01-.557-.328V8.887c0-.286.307-.466.557-.327l5.603 3.112z" />
                </svg>
                View Live Demo
              </a>
              <a
                href="https://github.com/lelemondev/lelemon"
                target="_blank"
                rel="noopener noreferrer"
                className="group inline-flex items-center gap-2 px-7 py-4 bg-white/80 backdrop-blur-sm border border-[#18181B]/10 hover:border-[#18181B]/20 hover:bg-white/95 text-[#18181B] font-semibold rounded-full shadow-[0_2px_8px_-2px_rgba(24,24,27,0.08)] hover:shadow-[0_4px_12px_-4px_rgba(24,24,27,0.12)] hover:-translate-y-0.5 transition-all duration-300 overflow-hidden"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                </svg>
                Star on GitHub
              </a>
            </div>
          </div>
        </section>

        {/* Features Split Section */}
        <section className="py-24 px-6 bg-white border-b border-[#18181B]/5">
          <div className="max-w-6xl mx-auto">
            <div className="grid md:grid-cols-2 gap-12 lg:gap-20 items-center">
              
              {/* Column 1: Features List */}
              <div className="space-y-6">
                <h2 className="text-4xl font-bold text-[#18181B] leading-snug">
                  Unify, Observe, and Control Your LLMs
                </h2>
                <p className="text-lg text-[#71717A] leading-relaxed">
                  Transform black-box AI agents into transparent systems. Lelemon provides complete visibility into every execution, allowing you to inspect prompts, tool calls, and model decisions effortlessly. Achieve zero latency with our asynchronous data ingestion and gain real-time cost control, from token counting to budget alerts. Simplify debugging, optimize performance, and manage your AI spend with confidence.
                </p>
              </div>

              {/* Column 2: Video (Centered with Gradient Background) */}
              <div className="flex items-center justify-center">
                <div className="relative group w-full max-w-2xl">
                  {/* Gradient Background Card */}
                  <div className="absolute inset-0 bg-gradient-to-br from-[#FACC15]/20 via-[#FEF08A]/30 to-white/40 rounded-3xl blur-2xl" />
                  
                  {/* Main Container */}
                  <div className="relative bg-gradient-to-br from-[#FACC15]/10 via-white to-[#FEF08A]/10 p-6 rounded-3xl shadow-2xl border border-[#FACC15]/20">
                    {/* Glow Effect */}
                    <div className="absolute -inset-1 bg-gradient-to-r from-[#FACC15]/30 via-[#FEF08A]/20 to-[#FACC15]/30 rounded-3xl blur-xl opacity-60 group-hover:opacity-100 transition-opacity duration-700" />
                    
                    <div className="relative rounded-2xl overflow-hidden shadow-xl border border-white/50 bg-[#18181B] aspect-video">
                    <iframe
                      className="absolute inset-0 w-full h-full"
                      src="https://player.cloudinary.com/embed/?cloud_name=dborlhema&public_id=lelemon-demo-ingles_e3gjvs&profile=cld-default"
                      allow="autoplay; fullscreen; encrypted-media; picture-in-picture"
                      allowFullScreen
                      title="Lelemon Demo"
                    ></iframe>

                    </div>
                  </div>
                  
                  {/* Decorative Elements */}
                  <div className="absolute -bottom-8 -right-8 w-32 h-32 bg-[#FACC15]/20 rounded-full blur-3xl -z-10" />
                  <div className="absolute -top-8 -left-8 w-40 h-40 bg-[#FEF08A]/15 rounded-full blur-3xl -z-10" />
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Product/Dev Section */}
        <section id="demo" className="py-24 px-6 bg-white">
          <div className="max-w-6xl mx-auto">
            <div className="grid lg:grid-cols-2 gap-16 lg:gap-20 items-center">
              {/* Left - Code */}
              <div className="order-2 lg:order-1 space-y-6">
                <div className="text-center lg:text-left mb-4">
                  <p className="text-sm text-[#71717A] font-medium">Drop-in integration for Node.js & Vercel AI SDK</p>
                </div>
                <CodeBlock />

              </div>

              {/* Right - Features */}
              <div className="order-1 lg:order-2 space-y-8">
                <div>
                  <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-2">
                    Built for developers
                  </h2>
                  <p className="text-[#71717A]">
                    No complications, straight to the code.
                  </p>
                </div>

                <ul className="space-y-5">
                  <FeatureCheck>
                    <strong className="text-[#18181B] font-semibold">Trace the complete flow:</strong> prompts, tool calls and outputs.
                  </FeatureCheck>
                  <FeatureCheck>
                    <strong className="text-[#18181B] font-semibold">Understand decisions and retries:</strong> see why the agent did what it did.
                  </FeatureCheck>
                  <FeatureCheck>
                    <strong className="text-[#18181B] font-semibold">Useful metrics:</strong> tokens and latency without friction.
                  </FeatureCheck>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* Stack Agnostic Section */}
        <section className="py-16 px-6 bg-white border-y border-[#18181B]/5">
          <div className="max-w-6xl mx-auto">
            <p className="text-center text-sm text-[#71717A] mb-8 font-medium">
              Works seamlessly with your favorite ingredients:
            </p>
            <div
              className="relative w-full overflow-hidden"
              style={{
                maskImage: 'linear-gradient(to right, transparent, black 20%, black 80%, transparent)',
              }}
            >
              <div className="flex w-max animate-infinite-scroll">
                {['OpenAI', 'Anthropic', 'OpenRouter', 'Vercel AI SDK', 'LangChain', 'Bedrock', 'Gemini'].flatMap((tech) => (
                  <div
                    key={tech}
                    className="px-5 py-2.5 rounded-lg bg-[#F4F4F5] text-[#3F3F46] text-sm font-semibold mx-4"
                  >
                    {tech}
                  </div>
                ))}
                {['OpenAI', 'Anthropic', 'OpenRouter', 'Vercel AI SDK', 'LangChain', 'Bedrock', 'Gemini'].flatMap((tech) => (
                  <div
                    key={`${tech}-2`}
                    className="px-5 py-2.5 rounded-lg bg-[#F4F4F5] text-[#3F3F46] text-sm font-semibold mx-4"
                  >
                    {tech}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* Footer */}
        <footer className="py-12 px-6">
          <div className="max-w-6xl mx-auto">
            <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
              <div className="flex items-center gap-3">
                <LemonIcon className="w-6 h-6" />
                <p className="text-sm text-[#71717A]">
                  Lelemon — Lightweight observability for generative AI agents. Built with Go, ClickHouse & Next.js.
                </p>
              </div>
              <nav className="flex flex-wrap items-center justify-center sm:justify-start gap-4 sm:gap-8 text-sm">
                <Link href="https://lelemondev.github.io/lelemondev-sdk/" target="_blank" className="text-[#71717A] hover:text-[#18181B] transition-colors font-medium">Docs</Link>
                <Link href="https://github.com/lelemondev/lelemon" target="_blank" className="text-[#71717A] hover:text-[#18181B] transition-colors font-medium">GitHub</Link>
              </nav>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
