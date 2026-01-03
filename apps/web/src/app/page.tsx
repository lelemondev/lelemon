import Link from 'next/link';

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
              <span className="text-primary-foreground font-bold text-lg">L</span>
            </div>
            <span className="font-bold text-xl">Lelemon</span>
          </div>
          <nav className="flex items-center gap-4">
            <Link
              href="/dashboard"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              Dashboard
            </Link>
            <Link
              href="https://github.com/lelemondev/lelemon"
              target="_blank"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              GitHub
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <main className="container mx-auto px-4">
        <section className="py-24 text-center">
          <h1 className="text-5xl font-bold tracking-tight mb-6">
            LLM Observability
            <br />
            <span className="text-muted-foreground">in 3 lines of code</span>
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto mb-8">
            Track your AI agents, debug tool calls, and understand costs.
            Works with OpenAI, Anthropic, Gemini, and AWS Bedrock.
          </p>
          <div className="flex gap-4 justify-center">
            <Link
              href="/dashboard"
              className="inline-flex items-center justify-center rounded-md bg-primary px-6 py-3 text-sm font-medium text-primary-foreground shadow hover:bg-primary/90 transition-colors"
            >
              Get Started
            </Link>
            <Link
              href="https://www.npmjs.com/package/@lelemondev/sdk"
              target="_blank"
              className="inline-flex items-center justify-center rounded-md border border-input bg-background px-6 py-3 text-sm font-medium shadow-sm hover:bg-accent hover:text-accent-foreground transition-colors"
            >
              npm install @lelemondev/sdk
            </Link>
          </div>
        </section>

        {/* Code Example */}
        <section className="py-12 max-w-3xl mx-auto">
          <div className="rounded-lg border bg-card overflow-hidden">
            <div className="flex items-center gap-2 px-4 py-3 border-b bg-muted/50">
              <div className="w-3 h-3 rounded-full bg-red-500" />
              <div className="w-3 h-3 rounded-full bg-yellow-500" />
              <div className="w-3 h-3 rounded-full bg-green-500" />
              <span className="ml-2 text-sm text-muted-foreground">agent.ts</span>
            </div>
            <pre className="p-4 text-sm overflow-x-auto">
              <code>{`import { trace } from '@lelemondev/sdk';

async function runAgent(userMessage: string) {
  const t = trace({ input: userMessage });

  try {
    const messages = [...];
    // ... your agent code ...
    await t.success(messages);
  } catch (error) {
    await t.error(error, messages);
  }
}`}</code>
            </pre>
          </div>
        </section>

        {/* Features */}
        <section className="py-16">
          <h2 className="text-3xl font-bold text-center mb-12">
            Everything you need to debug AI agents
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            <div className="p-6 rounded-lg border bg-card">
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                <svg className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="font-semibold mb-2">Zero Config</h3>
              <p className="text-sm text-muted-foreground">
                Auto-detects OpenAI, Anthropic, Gemini, and Bedrock message formats. Just pass your messages array.
              </p>
            </div>
            <div className="p-6 rounded-lg border bg-card">
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                <svg className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
              </div>
              <h3 className="font-semibold mb-2">Tool Call Tracing</h3>
              <p className="text-sm text-muted-foreground">
                See every tool call, its inputs, outputs, and duration. Debug agent loops with full visibility.
              </p>
            </div>
            <div className="p-6 rounded-lg border bg-card">
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                <svg className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <h3 className="font-semibold mb-2">Cost Analytics</h3>
              <p className="text-sm text-muted-foreground">
                Track token usage and costs per trace. Know exactly how much each agent run costs you.
              </p>
            </div>
          </div>
        </section>

        {/* CTA */}
        <section className="py-16 text-center">
          <h2 className="text-3xl font-bold mb-4">Ready to debug your AI agents?</h2>
          <p className="text-muted-foreground mb-8">
            Start tracing in under a minute. No credit card required.
          </p>
          <Link
            href="/dashboard"
            className="inline-flex items-center justify-center rounded-md bg-primary px-8 py-3 text-sm font-medium text-primary-foreground shadow hover:bg-primary/90 transition-colors"
          >
            Get Started Free
          </Link>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t py-8">
        <div className="container mx-auto px-4 text-center text-sm text-muted-foreground">
          <p>Built by <a href="https://github.com/lelemondev" className="underline hover:text-foreground">lelemondev</a></p>
        </div>
      </footer>
    </div>
  );
}
