'use client';

import { useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { useProject } from '@/lib/project-context';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface Trace {
  id: string;
  sessionId: string | null;
  userId: string | null;
  status: 'active' | 'completed' | 'error';
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: string;
  totalDurationMs: number;
  tags: string[] | null;
  createdAt: string;
}

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
  const { currentProject } = useProject();
  const [traces, setTraces] = useState<Trace[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Filters
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [tagFilter, setTagFilter] = useState<string>('all');

  // Fetch traces
  useEffect(() => {
    if (!currentProject) {
      setTraces([]);
      setIsLoading(false);
      return;
    }

    const fetchTraces = async () => {
      setIsLoading(true);
      try {
        const response = await fetch(`/api/v1/dashboard/traces?projectId=${currentProject.id}`);
        if (response.ok) {
          const data = await response.json();
          setTraces(data);
        }
      } catch (error) {
        console.error('Failed to fetch traces:', error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchTraces();
  }, [currentProject]);

  // Get unique tags from traces
  const allTags = useMemo(() => {
    const tagSet = new Set<string>();
    traces.forEach((trace) => {
      trace.tags?.forEach((tag) => tagSet.add(tag));
    });
    return Array.from(tagSet).sort();
  }, [traces]);

  // Filter traces
  const filteredTraces = useMemo(() => {
    return traces.filter((trace) => {
      // Search filter
      if (search) {
        const searchLower = search.toLowerCase();
        const matchesSession = trace.sessionId?.toLowerCase().includes(searchLower);
        const matchesUser = trace.userId?.toLowerCase().includes(searchLower);
        const matchesId = trace.id.toLowerCase().includes(searchLower);
        if (!matchesSession && !matchesUser && !matchesId) return false;
      }

      // Status filter
      if (statusFilter !== 'all' && trace.status !== statusFilter) return false;

      // Tag filter
      if (tagFilter !== 'all' && !trace.tags?.includes(tagFilter)) return false;

      return true;
    });
  }, [traces, search, statusFilter, tagFilter]);

  if (!currentProject) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-500/10 flex items-center justify-center mb-4">
          <svg className="w-8 h-8 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">No project selected</h3>
        <p className="text-zinc-500 dark:text-zinc-400 mb-4">Select or create a project to view traces.</p>
        <Link href="/dashboard/projects">
          <Button className="bg-amber-500 hover:bg-amber-600 text-zinc-900">
            Go to Projects
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Traces</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            View and analyze all your LLM traces.
          </p>
        </div>
        <div className="text-sm text-zinc-500 dark:text-zinc-400">
          {filteredTraces.length} of {traces.length} traces
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex-1 min-w-[200px] max-w-sm">
              <Input
                placeholder="Search by session, user, or trace ID..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full"
              />
            </div>

            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
                <SelectItem value="error">Error</SelectItem>
              </SelectContent>
            </Select>

            <Select value={tagFilter} onValueChange={setTagFilter}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="Tag" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Tags</SelectItem>
                {allTags.map((tag) => (
                  <SelectItem key={tag} value={tag}>
                    {tag}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {(search || statusFilter !== 'all' || tagFilter !== 'all') && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSearch('');
                  setStatusFilter('all');
                  setTagFilter('all');
                }}
              >
                Clear filters
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Traces Table */}
      <Card>
        <CardHeader>
          <CardTitle>All Traces</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="h-12 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse" />
              ))}
            </div>
          ) : filteredTraces.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-zinc-500 dark:text-zinc-400">
                {traces.length === 0
                  ? 'No traces yet. Start by sending traces from your application.'
                  : 'No traces match your filters.'}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[100px]">Time</TableHead>
                  <TableHead className="hidden md:table-cell">Session</TableHead>
                  <TableHead className="hidden lg:table-cell">User</TableHead>
                  <TableHead className="hidden lg:table-cell">Tags</TableHead>
                  <TableHead className="hidden sm:table-cell text-right">Spans</TableHead>
                  <TableHead className="text-right">Tokens</TableHead>
                  <TableHead className="text-right">Cost</TableHead>
                  <TableHead className="hidden sm:table-cell text-right">Duration</TableHead>
                  <TableHead className="text-center">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTraces.map((trace) => (
                  <TableRow key={trace.id} className="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                    <TableCell className="font-medium">
                      <Link
                        href={`/dashboard/traces/${trace.id}`}
                        className="text-amber-600 dark:text-amber-400 hover:underline"
                      >
                        {formatRelativeTime(trace.createdAt)}
                      </Link>
                    </TableCell>
                    <TableCell className="hidden md:table-cell font-mono text-xs text-zinc-600 dark:text-zinc-400">
                      {trace.sessionId?.slice(0, 16) || '-'}
                    </TableCell>
                    <TableCell className="hidden lg:table-cell text-zinc-600 dark:text-zinc-400">
                      {trace.userId || 'Anonymous'}
                    </TableCell>
                    <TableCell className="hidden lg:table-cell">
                      <div className="flex gap-1 flex-wrap">
                        {trace.tags?.map((tag) => (
                          <Badge
                            key={tag}
                            variant="secondary"
                            className="text-xs cursor-pointer hover:bg-zinc-200 dark:hover:bg-zinc-700"
                            onClick={() => setTagFilter(tag)}
                          >
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="hidden sm:table-cell text-right text-zinc-600 dark:text-zinc-400">
                      {trace.totalSpans}
                    </TableCell>
                    <TableCell className="text-right text-zinc-600 dark:text-zinc-400">
                      {trace.totalTokens.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right font-mono text-amber-600 dark:text-amber-400">
                      ${parseFloat(trace.totalCostUsd).toFixed(4)}
                    </TableCell>
                    <TableCell className="hidden sm:table-cell text-right text-zinc-600 dark:text-zinc-400">
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
                        className="text-xs"
                      >
                        {trace.status === 'completed' && (
                          <svg className="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                          </svg>
                        )}
                        {trace.status === 'error' && (
                          <svg className="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        )}
                        {trace.status === 'active' && (
                          <svg className="w-3 h-3 mr-1 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                        )}
                        {trace.status}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
