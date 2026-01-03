'use client';

import { use } from 'react';
import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

interface Span {
  id: string;
  type: 'llm' | 'tool' | 'retrieval' | 'custom';
  name: string;
  model?: string;
  provider?: string;
  inputTokens?: number;
  outputTokens?: number;
  durationMs: number;
  costUsd?: string;
  status: 'success' | 'error' | 'pending';
  errorMessage?: string;
  input?: unknown;
  output?: unknown;
}

// Mock data
const mockTrace = {
  id: '1a2b3c4d',
  sessionId: 'session-abc123',
  userId: 'user-001',
  status: 'completed' as const,
  totalSpans: 3,
  totalTokens: 1847,
  totalCostUsd: '0.0234',
  totalDurationMs: 2340,
  tags: ['sales', 'demo'],
  metadata: {
    source: 'whatsapp',
    campaign: 'black-friday',
    region: 'latam',
  },
  createdAt: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
  spans: [
    {
      id: 'span-1',
      type: 'llm' as const,
      name: 'openai.chat',
      model: 'gpt-4-turbo',
      provider: 'openai',
      inputTokens: 1234,
      outputTokens: 456,
      durationMs: 1200,
      costUsd: '0.0156',
      status: 'success' as const,
      input: {
        messages: [
          { role: 'system', content: 'You are a helpful sales assistant...' },
          { role: 'user', content: 'Tell me about pricing' },
        ],
      },
      output: {
        content: 'Our pricing starts at $29/month for the starter plan...',
      },
    },
    {
      id: 'span-2',
      type: 'tool' as const,
      name: 'search_documents',
      durationMs: 340,
      status: 'success' as const,
      input: { query: 'pricing plans features' },
      output: { results: ['Plan A: $29/mo', 'Plan B: $99/mo', 'Enterprise: Custom'] },
    },
    {
      id: 'span-3',
      type: 'llm' as const,
      name: 'openai.chat',
      model: 'gpt-4-turbo',
      provider: 'openai',
      inputTokens: 89,
      outputTokens: 68,
      durationMs: 800,
      costUsd: '0.0078',
      status: 'success' as const,
      input: {
        messages: [
          { role: 'user', content: 'Which plan is best for a small startup?' },
        ],
      },
      output: {
        content: 'For a small startup, I recommend starting with our Starter plan at $29/month...',
      },
    },
  ] as Span[],
};

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function SpanTypeIcon({ type }: { type: Span['type'] }) {
  const icons = {
    llm: '🤖',
    tool: '🔧',
    retrieval: '🔍',
    custom: '⚙️',
  };
  return <span>{icons[type]}</span>;
}

export default function TraceDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const trace = mockTrace; // In real app, fetch by id

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/dashboard/traces">
            <Button variant="ghost" size="sm">
              ← Back
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              Trace {id.slice(0, 8)}
            </h1>
            <p className="text-sm text-muted-foreground">
              Session: {trace.sessionId} | User: {trace.userId}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {trace.tags.map((tag) => (
            <Badge key={tag} variant="secondary">
              {tag}
            </Badge>
          ))}
          <Badge
            variant={trace.status === 'completed' ? 'default' : 'destructive'}
          >
            {trace.status}
          </Badge>
        </div>
      </div>

      {/* Summary */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Duration
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatDuration(trace.totalDurationMs)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Spans
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{trace.totalSpans}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Tokens
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {trace.totalTokens.toLocaleString()}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Cost
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${trace.totalCostUsd}</div>
          </CardContent>
        </Card>
      </div>

      {/* Spans Timeline */}
      <Card>
        <CardHeader>
          <CardTitle>Spans</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {trace.spans.map((span, index) => (
              <div
                key={span.id}
                className="border rounded-lg p-4 space-y-3"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-xl">
                      <SpanTypeIcon type={span.type} />
                    </span>
                    <div>
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="uppercase text-xs">
                          {span.type}
                        </Badge>
                        <span className="font-medium">{span.name}</span>
                      </div>
                      {span.model && (
                        <p className="text-sm text-muted-foreground">
                          Model: {span.model} ({span.provider})
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    {span.inputTokens && (
                      <span>↑ {span.inputTokens} tokens</span>
                    )}
                    {span.outputTokens && (
                      <span>↓ {span.outputTokens} tokens</span>
                    )}
                    <span>{formatDuration(span.durationMs)}</span>
                    {span.costUsd && <span>${span.costUsd}</span>}
                    <Badge
                      variant={
                        span.status === 'success'
                          ? 'default'
                          : span.status === 'error'
                          ? 'destructive'
                          : 'secondary'
                      }
                    >
                      {span.status}
                    </Badge>
                  </div>
                </div>

                {/* Input/Output */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">
                      Input
                    </p>
                    <pre className="text-xs bg-zinc-50 dark:bg-zinc-900 p-2 rounded overflow-auto max-h-32">
                      {JSON.stringify(span.input, null, 2)}
                    </pre>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">
                      Output
                    </p>
                    <pre className="text-xs bg-zinc-50 dark:bg-zinc-900 p-2 rounded overflow-auto max-h-32">
                      {JSON.stringify(span.output, null, 2)}
                    </pre>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Metadata */}
      <Card>
        <CardHeader>
          <CardTitle>Metadata</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="text-sm bg-zinc-50 dark:bg-zinc-900 p-4 rounded overflow-auto">
            {JSON.stringify(trace.metadata, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
