'use client';

import { Suspense, useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useProject } from '@/lib/project-context';
import { dashboardAPI, Trace } from '@/lib/api';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { TracesList, TraceDetail } from '@/components/traces';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

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

function TracesPageContent() {
  const { currentProject } = useProject();
  const searchParams = useSearchParams();
  const [traces, setTraces] = useState<Trace[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);

  // Get sessionId from URL if present (for filtering by session)
  const sessionIdFromUrl = searchParams.get('sessionId');

  // Filters
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [sessionFilter, setSessionFilter] = useState<string>(sessionIdFromUrl || 'all');

  // Polling interval in ms
  const POLLING_INTERVAL = 5000;

  // Fetch traces when project ID changes, with polling
  const projectId = currentProject?.id;
  useEffect(() => {
    if (!projectId) {
      setTraces([]);
      setIsLoading(false);
      return;
    }

    let isMounted = true;

    const fetchTraces = async (showLoading = false) => {
      if (showLoading) setIsLoading(true);
      try {
        const result = await dashboardAPI.getTraces(projectId, { limit: 100 });
        if (isMounted) {
          setTraces(result.data);
        }
      } catch (error) {
        console.error('Failed to fetch traces:', error);
      } finally {
        if (isMounted && showLoading) {
          setIsLoading(false);
        }
      }
    };

    // Initial fetch with loading state
    fetchTraces(true);

    // Set up polling
    const intervalId = setInterval(() => {
      fetchTraces(false);
    }, POLLING_INTERVAL);

    // Cleanup on unmount
    return () => {
      isMounted = false;
      clearInterval(intervalId);
    };
  }, [projectId]);

  // Get unique sessions from traces
  const allSessions = useMemo(() => {
    const sessionSet = new Set<string>();
    traces.forEach((trace) => {
      if (trace.sessionId) sessionSet.add(trace.sessionId);
    });
    return Array.from(sessionSet).sort();
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

      // Session filter
      if (sessionFilter !== 'all' && trace.sessionId !== sessionFilter) return false;

      return true;
    });
  }, [traces, search, statusFilter, sessionFilter]);

  // Clear selection if selected trace is filtered out
  useEffect(() => {
    if (selectedTraceId && !filteredTraces.find(t => t.id === selectedTraceId)) {
      setSelectedTraceId(filteredTraces[0]?.id ?? null);
    }
  }, [filteredTraces, selectedTraceId]);

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

  const hasActiveFilters = search || statusFilter !== 'all' || sessionFilter !== 'all';

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex-shrink-0 mb-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">
              {sessionFilter !== 'all' ? 'Session Traces' : 'Traces'}
            </h1>
            <p className="text-zinc-500 dark:text-zinc-400 text-sm mt-0.5">
              {sessionFilter !== 'all' ? (
                <>
                  Viewing traces for session{' '}
                  <code className="px-1 py-0.5 bg-zinc-100 dark:bg-zinc-800 rounded text-xs font-mono">
                    {sessionFilter.length > 20 ? `${sessionFilter.slice(0, 20)}...` : sessionFilter}
                  </code>
                </>
              ) : (
                'View and analyze all your LLM traces.'
              )}
            </p>
          </div>
          {sessionFilter !== 'all' && (
            <Link href="/dashboard/sessions">
              <Button variant="outline" size="sm">
                <svg className="w-4 h-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
                </svg>
                All Sessions
              </Button>
            </Link>
          )}
        </div>
      </div>

      {/* Main Content */}
      {selectedTraceId ? (
        /* Master-Detail Layout when trace is selected */
        <div className="flex-1 flex gap-4 min-h-0">
          {/* Left Panel: Traces List (compact) */}
          <div className="w-72 flex-shrink-0 flex flex-col border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden">
            {/* Compact Filters */}
            <div className="flex-shrink-0 p-2 border-b border-zinc-200 dark:border-zinc-700 space-y-2">
              <Input
                placeholder="Search..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="h-7 text-xs"
              />
              <div className="flex gap-1">
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                  <SelectTrigger className="h-7 text-xs flex-1">
                    <SelectValue placeholder="Status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="completed">Done</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* List Header */}
            <div className="flex-shrink-0 px-2 py-1.5 border-b border-zinc-200 dark:border-zinc-700">
              <span className="text-xs text-zinc-500 dark:text-zinc-400">
                {filteredTraces.length} traces
              </span>
            </div>

            {/* Scrollable List */}
            <div className="flex-1 overflow-auto">
              <TracesList
                traces={filteredTraces}
                selectedTraceId={selectedTraceId}
                onSelectTrace={setSelectedTraceId}
                isLoading={isLoading}
              />
            </div>
          </div>

          {/* Right Panel: Trace Detail */}
          <div className="flex-1 border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden min-w-0">
            <TraceDetail
              traceId={selectedTraceId}
              onClose={() => setSelectedTraceId(null)}
            />
          </div>
        </div>
      ) : (
        /* Full-width Table when no trace selected */
        <div className="flex-1 flex flex-col border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden">
          {/* Filters Row */}
          <div className="flex-shrink-0 p-4 border-b border-zinc-200 dark:border-zinc-700">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex-1 min-w-[200px] max-w-sm">
                <Input
                  placeholder="Search by session, user, or trace ID..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="h-9"
                />
              </div>

              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-[130px] h-9">
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Status</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="error">Error</SelectItem>
                </SelectContent>
              </Select>

              {allSessions.length > 0 && (
                <Select value={sessionFilter} onValueChange={setSessionFilter}>
                  <SelectTrigger className="w-[160px] h-9">
                    <SelectValue placeholder="Session" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Sessions</SelectItem>
                    {allSessions.map((session) => (
                      <SelectItem key={session} value={session}>
                        {session.length > 16 ? `${session.slice(0, 16)}...` : session}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}

              {hasActiveFilters && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setSearch('');
                    setStatusFilter('all');
                    setSessionFilter('all');
                  }}
                >
                  Clear filters
                </Button>
              )}

              <div className="ml-auto text-sm text-zinc-500 dark:text-zinc-400">
                {filteredTraces.length} traces
              </div>
            </div>
          </div>

          {/* Table */}
          <div className="flex-1 overflow-auto">
            {isLoading ? (
              <div className="p-4 space-y-3">
                {[1, 2, 3, 4, 5].map((i) => (
                  <div key={i} className="h-12 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse" />
                ))}
              </div>
            ) : filteredTraces.length === 0 ? (
              <div className="flex items-center justify-center h-64">
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
                    <TableHead>Trace ID</TableHead>
                    <TableHead className="hidden md:table-cell">Session</TableHead>
                    <TableHead className="hidden lg:table-cell">User</TableHead>
                    <TableHead className="text-right">Spans</TableHead>
                    <TableHead className="text-right">Tokens</TableHead>
                    <TableHead className="text-right">Cost</TableHead>
                    <TableHead className="hidden sm:table-cell text-right">Duration</TableHead>
                    <TableHead className="text-center">Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredTraces.map((trace) => (
                    <TableRow
                      key={trace.id}
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                      onClick={() => setSelectedTraceId(trace.id)}
                    >
                      <TableCell className="font-medium text-amber-600 dark:text-amber-400">
                        {formatRelativeTime(trace.createdAt)}
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {trace.id.slice(0, 12)}...
                      </TableCell>
                      <TableCell className="hidden md:table-cell font-mono text-xs text-zinc-500">
                        {trace.sessionId ? `${trace.sessionId.slice(0, 12)}...` : '-'}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell text-zinc-600 dark:text-zinc-400">
                        {trace.userId || 'Anonymous'}
                      </TableCell>
                      <TableCell className="text-right text-zinc-600 dark:text-zinc-400">
                        {trace.totalSpans}
                      </TableCell>
                      <TableCell className="text-right text-zinc-600 dark:text-zinc-400">
                        {trace.totalTokens.toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right font-mono text-amber-600 dark:text-amber-400">
                        ${trace.totalCostUsd.toFixed(4)}
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
                          {trace.status === 'active' && (
                            <span className="relative flex h-2 w-2 mr-1">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                              <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                            </span>
                          )}
                          {trace.status}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function TracesPageFallback() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Traces</h1>
        <p className="text-zinc-500 dark:text-zinc-400 mt-1">
          View and manage your LLM trace data.
        </p>
      </div>
      <div className="h-96 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
    </div>
  );
}

export default function TracesPage() {
  return (
    <Suspense fallback={<TracesPageFallback />}>
      <TracesPageContent />
    </Suspense>
  );
}
