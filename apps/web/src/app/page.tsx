'use client';

import { useState } from 'react';
import Link from 'next/link';

function LemonIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 64 64" fill="none">
      <ellipse cx="32" cy="34" rx="24" ry="26" fill="#FFE566" />
      <ellipse cx="32" cy="34" rx="24" ry="26" fill="url(#lemon-gradient)" />
      <ellipse cx="28" cy="30" rx="16" ry="18" fill="#FFF9E0" opacity="0.4" />
      <path d="M32 8C32 8 28 4 32 2C36 4 32 8 32 8Z" fill="#4ADE80" />
      <path d="M30 8C28 6 24 7 24 7" stroke="#4ADE80" strokeWidth="2" strokeLinecap="round" />
      <defs>
        <linearGradient id="lemon-gradient" x1="8" y1="8" x2="56" y2="60">
          <stop stopColor="#FFE566" />
          <stop offset="1" stopColor="#FFD84D" />
        </linearGradient>
      </defs>
    </svg>
  );
}


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
      {/* Glow effect */}
      <div className="absolute -inset-1 bg-gradient-to-r from-[#FFD84D]/20 via-[#FFE566]/10 to-[#FFD84D]/20 rounded-2xl blur-xl opacity-0 group-hover:opacity-100 transition-opacity duration-500" />

      <div className="relative rounded-2xl bg-[#1B1B1B] overflow-hidden shadow-2xl border border-white/5">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/10 bg-gradient-to-r from-[#252525] to-[#1B1B1B]">
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
        <pre className="p-6 text-[13px] font-mono leading-7 overflow-x-auto">
          <code>
            <span className="text-[#C678DD]">import</span>
            <span className="text-white/90">{' { '}</span>
            <span className="text-[#E5C07B]">init</span>
            <span className="text-white/90">{', '}</span>
            <span className="text-[#E5C07B]">observe</span>
            <span className="text-white/90">{' } '}</span>
            <span className="text-[#C678DD]">from</span>
            <span className="text-[#98C379]">{" '@lelemondev/sdk'"}</span>
            <span className="text-white/30">;</span>
            {'\n'}
            <span className="text-[#C678DD]">import</span>
            <span className="text-[#E5C07B]"> OpenAI </span>
            <span className="text-[#C678DD]">from</span>
            <span className="text-[#98C379]">{" 'openai'"}</span>
            <span className="text-white/30">;</span>
            {'\n\n'}
            <span className="text-[#61AFEF]">init</span>
            <span className="text-white/90">{'({ '}</span>
            <span className="text-[#E06C75]">apiKey</span>
            <span className="text-white/90">: process.env.</span>
            <span className="text-[#E5C07B]">LELEMON_API_KEY</span>
            <span className="text-white/90">{' });'}</span>
            {'\n\n'}
            <span className="text-[#C678DD]">const</span>
            <span className="text-[#E06C75]"> openai</span>
            <span className="text-white/90"> = </span>
            <span className="text-[#61AFEF]">observe</span>
            <span className="text-white/90">(</span>
            <span className="text-[#C678DD]">new</span>
            <span className="text-[#E5C07B]"> OpenAI</span>
            <span className="text-white/90">());</span>
            {'\n\n'}
            <span className="text-[#C678DD]">const</span>
            <span className="text-[#E06C75]"> res</span>
            <span className="text-white/90"> = </span>
            <span className="text-[#C678DD]">await</span>
            <span className="text-[#E06C75]"> openai</span>
            <span className="text-white/90">.chat.completions.</span>
            <span className="text-[#61AFEF]">create</span>
            <span className="text-white/90">{'({'}</span>
            {'\n'}
            <span className="text-white/90">{'  '}</span>
            <span className="text-[#E06C75]">model</span>
            <span className="text-white/90">: </span>
            <span className="text-[#98C379]">{"'gpt-4'"}</span>
            <span className="text-white/90">,</span>
            {'\n'}
            <span className="text-white/90">{'  '}</span>
            <span className="text-[#E06C75]">messages</span>
            <span className="text-white/90">{': [{ '}</span>
            <span className="text-[#E06C75]">role</span>
            <span className="text-white/90">: </span>
            <span className="text-[#98C379]">{"'user'"}</span>
            <span className="text-white/90">{', '}</span>
            <span className="text-[#E06C75]">content</span>
            <span className="text-white/90">: </span>
            <span className="text-[#98C379]">{"'Hello!'"}</span>
            <span className="text-white/90">{' }],'}</span>
            {'\n'}
            <span className="text-white/90">{'});'}</span>
          </code>
        </pre>
      </div>
    </div>
  );
}

function FeatureCheck({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-3 group">
      <div className="mt-1 w-5 h-5 rounded-full bg-gradient-to-br from-[#FFE566] to-[#FFD84D] flex items-center justify-center shadow-[0_2px_8px_-2px_rgba(255,216,77,0.5)] group-hover:shadow-[0_4px_12px_-2px_rgba(255,216,77,0.6)] transition-shadow">
        <svg className="w-3 h-3 text-[#1B1B1B]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
        </svg>
      </div>
      <span className="text-[#1B1B1B]/80 leading-relaxed">{children}</span>
    </li>
  );
}

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-[#FFFDF8] text-[#1B1B1B] overflow-hidden">
      {/* Background elements */}
      <div className="fixed inset-0 pointer-events-none">
        {/* Subtle grid */}
        <div className="absolute inset-0 opacity-[0.015]" style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M0 0h60v60H0z' fill='none' stroke='%231B1B1B' stroke-width='1'/%3E%3C/svg%3E")`,
        }} />
        {/* Gradient blobs */}
        <div className="absolute top-0 right-0 w-[600px] h-[600px] bg-gradient-to-bl from-[#FFD84D]/20 via-[#FFE566]/10 to-transparent rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 w-[400px] h-[400px] bg-gradient-to-tr from-[#FFD84D]/10 to-transparent rounded-full blur-3xl" />
      </div>

      {/* Content */}
      <div className="relative">
        {/* Header */}
        <header className="fixed top-0 left-0 right-0 z-50 bg-[#FFFDF8]/80 backdrop-blur-md border-b border-[#1B1B1B]/5">
          <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
            <Link href="/" className="flex items-center gap-2.5 group">
              <div className="relative">
                <LemonIcon className="w-9 h-9 transition-transform group-hover:rotate-12 group-hover:scale-110" />
              </div>
              <span className="font-bold text-xl tracking-tight">Lelemon</span>
            </Link>
            <Link
              href="/login"
              className="text-sm px-5 py-2.5 rounded-full bg-[#FFD84D] text-[#1B1B1B] font-semibold hover:bg-[#F5C800] transition-all shadow-[0_2px_10px_-2px_rgba(255,216,77,0.4)]"
            >
              Sign in
            </Link>
          </div>
        </header>

        {/* Hero Section */}
        <section className="pt-32 pb-24 px-6">
          <div className="max-w-4xl mx-auto text-center">
            {/* Badge */}
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[#FFD84D]/10 border border-[#FFD84D]/20 mb-8">
              <span className="text-sm">🍋</span>
              <span className="text-sm font-medium text-[#B8860B]">LLM Observability</span>
            </div>

            <h1 className="text-5xl sm:text-6xl font-extrabold tracking-tight leading-[1.1] mb-6">
              Trace your LLMs
              <br />
              <span className="relative inline-block">
                with a twist
                <svg className="absolute -bottom-2 left-0 w-full h-3 text-[#FFD84D]" viewBox="0 0 200 12" fill="none">
                  <path d="M2 10C50 4 150 4 198 10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
                </svg>
              </span>
              {' '}
              <span className="inline-block animate-bounce-slow">🍋</span>
            </h1>

            <p className="text-xl text-[#1B1B1B]/60 leading-relaxed max-w-2xl mx-auto mb-10">
              Lelemon te muestra qué hacen tus agentes: prompts, decisiones y métricas en tiempo real.
              <span className="text-[#1B1B1B] font-medium"> Claro, rápido y sin enredos.</span>
            </p>

            <div className="flex justify-center">
              <a
                href="#demo"
                className="group inline-flex items-center gap-2 px-7 py-4 bg-[#FFD84D] text-[#1B1B1B] font-semibold rounded-full hover:bg-[#F5C800] transition-all shadow-[0_4px_20px_-4px_rgba(255,216,77,0.5)] hover:shadow-[0_8px_30px_-4px_rgba(255,216,77,0.6)] hover:-translate-y-0.5"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.91 11.672a.375.375 0 010 .656l-5.603 3.113a.375.375 0 01-.557-.328V8.887c0-.286.307-.466.557-.327l5.603 3.112z" />
                </svg>
                View Demo
              </a>
            </div>
          </div>
        </section>

        {/* Product/Dev Section */}
        <section id="demo" className="py-24 px-6 bg-white border-y border-[#1B1B1B]/5">
          <div className="max-w-6xl mx-auto">
            <div className="grid lg:grid-cols-2 gap-16 lg:gap-20 items-center">
              {/* Left - Code */}
              <div className="order-2 lg:order-1 space-y-6">
                <CodeBlock />
                <div className="flex items-center gap-2 text-sm text-[#1B1B1B]/50">
                  <span className="font-medium">Providers:</span>
                  <div className="flex gap-2">
                    {['OpenAI', 'Anthropic', 'Bedrock', 'Gemini'].map((name) => (
                      <span key={name} className="px-3 py-1 rounded-full bg-[#F9F9FB] text-[#1B1B1B]/70 text-xs font-medium">
                        {name}
                      </span>
                    ))}
                  </div>
                </div>
              </div>

              {/* Right - Features */}
              <div className="order-1 lg:order-2 space-y-8">
                <div>
                  <h2 className="text-4xl font-bold tracking-tight mb-2">
                    Built for developers
                  </h2>
                  <p className="text-[#1B1B1B]/50">
                    Sin complicaciones, directo al código.
                  </p>
                </div>

                <ul className="space-y-5">
                  <FeatureCheck>
                    <strong className="text-[#1B1B1B] font-semibold">Tracea el flujo completo:</strong> prompts, tool calls y outputs.
                  </FeatureCheck>
                  <FeatureCheck>
                    <strong className="text-[#1B1B1B] font-semibold">Entiende decisiones y retries:</strong> ve por qué el agente hizo lo que hizo.
                  </FeatureCheck>
                  <FeatureCheck>
                    <strong className="text-[#1B1B1B] font-semibold">Métricas útiles:</strong> tokens y latencia sin fricción.
                  </FeatureCheck>
                </ul>
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
                <p className="text-sm text-[#1B1B1B]/50">
                  © Lelemon — Observability for generative agents.
                </p>
              </div>
              <nav className="flex items-center gap-8 text-sm">
                <Link href="https://lelemondev.github.io/lelemondev-sdk/" target="_blank" className="text-[#1B1B1B]/50 hover:text-[#1B1B1B] transition-colors font-medium">Docs</Link>
                <Link href="https://github.com/lelemondev/lelemondev-sdk" target="_blank" className="text-[#1B1B1B]/50 hover:text-[#1B1B1B] transition-colors font-medium">GitHub</Link>
                <Link href="#" className="text-[#1B1B1B]/50 hover:text-[#1B1B1B] transition-colors font-medium">Privacy</Link>
                <Link href="#" className="text-[#1B1B1B]/50 hover:text-[#1B1B1B] transition-colors font-medium">Terms</Link>
              </nav>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
