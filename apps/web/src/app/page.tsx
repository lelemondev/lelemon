'use client';

import { useState } from 'react';
import Link from 'next/link';
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
      {/* Glow effect - más sutil */}
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
        <pre className="p-4 sm:p-6 text-[11px] sm:text-[13px] font-mono leading-6 sm:leading-7 overflow-x-auto">
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
        {/* Gradient blobs - más sutiles */}
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
            <Link
              href="/login"
              className="text-sm px-5 py-2.5 rounded-full bg-[#FACC15] text-[#18181B] font-semibold hover:bg-[#EAB308] transition-all shadow-[0_2px_8px_-2px_rgba(250,204,21,0.35)]"
            >
              Sign in
            </Link>
          </div>
        </header>

        {/* Hero Section */}
        <section className="pt-32 pb-20 px-6">
          <div className="max-w-4xl mx-auto text-center">
            {/* Badge */}
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[#FACC15]/10 border border-[#FACC15]/20 mb-8">
              <span className="text-sm">🍋</span>
              <span className="text-sm font-medium text-[#A16207]">LLM Observability</span>
            </div>

            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight leading-[1.1] mb-6">
              Trace your LLMs
              <br />
              <span className="relative inline-block">
                with a twist
                <svg className="absolute -bottom-2 left-0 w-full h-3 text-[#FACC15]" viewBox="0 0 200 12" fill="none">
                  <path d="M2 10C50 4 150 4 198 10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
                </svg>
              </span>
              {' '}
              <span className="inline-block animate-bounce-slow">🍋</span>
            </h1>

            <p className="text-xl text-[#3F3F46] leading-relaxed max-w-2xl mx-auto mb-8">
              Lelemon te muestra qué hacen tus agentes: prompts, decisiones y métricas en tiempo real.
              <span className="text-[#18181B] font-medium"> Claro, rápido y sin enredos.</span>
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
                &lt;2KB gzipped
              </span>
              <span className="flex items-center gap-1.5 text-[#3F3F46]">
                <svg className="w-4 h-4 text-[#FACC15]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                No overhead
              </span>
            </div>

            <div className="flex justify-center">
              <a
                href="#demo"
                className="group inline-flex items-center gap-2 px-7 py-4 bg-[#FACC15] text-[#18181B] font-semibold rounded-full hover:bg-[#EAB308] transition-all shadow-[0_2px_12px_-4px_rgba(250,204,21,0.4)] hover:shadow-[0_4px_16px_-4px_rgba(250,204,21,0.5)] hover:-translate-y-0.5"
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
        <section id="demo" className="py-24 px-6 bg-white border-y border-[#18181B]/5">
          <div className="max-w-6xl mx-auto">
            <div className="grid lg:grid-cols-2 gap-16 lg:gap-20 items-center">
              {/* Left - Code */}
              <div className="order-2 lg:order-1 space-y-6">
                <CodeBlock />
                <div className="flex flex-col sm:flex-row sm:items-center gap-2 text-sm text-[#71717A]">
                  <span className="font-medium">Providers:</span>
                  <div className="flex flex-wrap gap-2">
                    {['OpenAI', 'Anthropic', 'Bedrock', 'Gemini'].map((name) => (
                      <span key={name} className="px-3 py-1 rounded-full bg-[#F4F4F5] text-[#3F3F46] text-xs font-medium">
                        {name}
                      </span>
                    ))}
                  </div>
                </div>
              </div>

              {/* Right - Features */}
              <div className="order-1 lg:order-2 space-y-8">
                <div>
                  <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-2">
                    Built for developers
                  </h2>
                  <p className="text-[#71717A]">
                    Sin complicaciones, directo al código.
                  </p>
                </div>

                <ul className="space-y-5">
                  <FeatureCheck>
                    <strong className="text-[#18181B] font-semibold">Tracea el flujo completo:</strong> prompts, tool calls y outputs.
                  </FeatureCheck>
                  <FeatureCheck>
                    <strong className="text-[#18181B] font-semibold">Entiende decisiones y retries:</strong> ve por qué el agente hizo lo que hizo.
                  </FeatureCheck>
                  <FeatureCheck>
                    <strong className="text-[#18181B] font-semibold">Métricas útiles:</strong> tokens y latencia sin fricción.
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
                <p className="text-sm text-[#71717A]">
                  © Lelemon — Observability for generative agents.
                </p>
              </div>
              <nav className="flex flex-wrap items-center justify-center sm:justify-start gap-4 sm:gap-8 text-sm">
                <Link href="https://lelemondev.github.io/lelemondev-sdk/" target="_blank" className="text-[#71717A] hover:text-[#18181B] transition-colors font-medium">Docs</Link>
                <Link href="https://github.com/lelemondev/lelemondev-sdk" target="_blank" className="text-[#71717A] hover:text-[#18181B] transition-colors font-medium">GitHub</Link>
                <Link href="#" className="text-[#71717A] hover:text-[#18181B] transition-colors font-medium">Privacy</Link>
                <Link href="#" className="text-[#71717A] hover:text-[#18181B] transition-colors font-medium">Terms</Link>
              </nav>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
