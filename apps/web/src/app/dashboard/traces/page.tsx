'use client';

import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

// Mock data
const mockTraces = [
  {
    id: '1a2b3c4d',
    sessionId: 'session-abc123',
    userId: 'user-001',
    status: 'completed' as const,
    totalSpans: 3,
    totalTokens: 1847,
    totalCostUsd: '0.0234',
    totalDurationMs: 2340,
    tags: ['sales', 'demo'],
    createdAt: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
  },
  {
    id: '2b3c4d5e',
    sessionId: 'session-def456',
    userId: 'user-002',
    status: 'completed' as const,
    totalSpans: 5,
    totalTokens: 3241,
    totalCostUsd: '0.0412',
    totalDurationMs: 4120,
    tags: ['support'],
    createdAt: new Date(Date.now() - 1000 * 60 * 12).toISOString(),
  },
  {
    id: '3c4d5e6f',
    sessionId: 'session-ghi789',
    userId: 'user-001',
    status: 'error' as const,
    totalSpans: 2,
    totalTokens: 892,
    totalCostUsd: '0.0089',
    totalDurationMs: 1230,
    tags: ['sales'],
    createdAt: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
  },
  {
    id: '4d5e6f7g',
    sessionId: 'session-jkl012',
    userId: 'user-003',
    status: 'completed' as const,
    totalSpans: 4,
    totalTokens: 2156,
    totalCostUsd: '0.0298',
    totalDurationMs: 3450,
    tags: ['onboarding', 'demo'],
    createdAt: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
  },
  {
    id: '5e6f7g8h',
    sessionId: 'session-mno345',
    userId: null,
    status: 'active' as const,
    totalSpans: 1,
    totalTokens: 234,
    totalCostUsd: '0.0023',
    totalDurationMs: 0,
    tags: [],
    createdAt: new Date(Date.now() - 1000 * 60 * 2).toISOString(),
  },
];

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 1000 / 60);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
  return `${Math.floor(diffMins / 1440)}d ago`;
}

function formatDuration(ms: number): string {
  if (ms === 0) return '-';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export default function TracesPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Traces</h1>
          <p className="text-muted-foreground">
            View and analyze all your LLM traces.
          </p>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <Input
              placeholder="Search by session ID or user ID..."
              className="max-w-sm"
            />
            <Button variant="outline">Filter</Button>
          </div>
        </CardContent>
      </Card>

      {/* Traces Table */}
      <Card>
        <CardHeader>
          <CardTitle>All Traces</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[100px]">Time</TableHead>
                <TableHead>Session</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Tags</TableHead>
                <TableHead className="text-right">Spans</TableHead>
                <TableHead className="text-right">Tokens</TableHead>
                <TableHead className="text-right">Cost</TableHead>
                <TableHead className="text-right">Duration</TableHead>
                <TableHead className="text-center">Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {mockTraces.map((trace) => (
                <TableRow key={trace.id}>
                  <TableCell className="font-medium">
                    <Link
                      href={`/dashboard/traces/${trace.id}`}
                      className="hover:underline"
                    >
                      {formatRelativeTime(trace.createdAt)}
                    </Link>
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {trace.sessionId?.slice(0, 16) || '-'}
                  </TableCell>
                  <TableCell>{trace.userId || 'Anonymous'}</TableCell>
                  <TableCell>
                    <div className="flex gap-1 flex-wrap">
                      {trace.tags.map((tag) => (
                        <Badge key={tag} variant="secondary" className="text-xs">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-right">{trace.totalSpans}</TableCell>
                  <TableCell className="text-right">
                    {trace.totalTokens.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    ${trace.totalCostUsd}
                  </TableCell>
                  <TableCell className="text-right">
                    {formatDuration(trace.totalDurationMs)}
                  </TableCell>
                  <TableCell className="text-center">
                    <Badge
                      variant={
                        trace.status === 'completed'
                          ? 'default'
                          : trace.status === 'error'
                          ? 'destructive'
                          : 'secondary'
                      }
                    >
                      {trace.status === 'completed' && '✓'}
                      {trace.status === 'error' && '✗'}
                      {trace.status === 'active' && '⟳'}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
