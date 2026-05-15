'use client';

import { Suspense, useState, useEffect, useMemo, useCallback } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
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
import { TagsFilter, DateRangeFilter } from '@/components/filters';

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
  const router = useRouter();
  const [traces, setTraces] = useState<Trace[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);

  // Get sessionId from URL if present (for filtering by session)
  const sessionIdFromUrl = searchParams.get('sessionId');
  // promptVersionId comes from the version-detail "X traces" payoff link.
  // Drives both the API filter and a visible banner that the user can clear.
  const promptVersionIdFromUrl = searchParams.get('promptVersionId');

  // Filters
  const [nameSearch, setNameSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [sessionFilter, setSessionFilter] = useState<string>(sessionIdFromUrl || 'all');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);

  // Polling interval in ms
  const POLLING_INTERVAL = 5000;

  // Build filter params for API
  const buildFilterParams = useCallback(() => {
    const params: {
      name?: string;
      status?: string;
      sessionId?: string;
      tags?: string[];
      promptVersionId?: string;
      from?: string;
      to?: string;
      limit: number;
    } = { limit: 100 };

    if (nameSearch) params.name = nameSearch;
    if (statusFilter !== 'all') params.status = statusFilter;
    if (sessionFilter !== 'all') params.sessionId = sessionFilter;
    if (selectedTags.length > 0) params.tags = selectedTags;
    if (promptVersionIdFromUrl) params.promptVersionId = promptVersionIdFromUrl;
    if (dateFrom) params.from = dateFrom.toISOString();
    if (dateTo) params.to = dateTo.toISOString();

    return params;
  }, [nameSearch, statusFilter, sessionFilter, selectedTags, promptVersionIdFromUrl, dateFrom, dateTo]);

  // Drop only the promptVersionId param from the URL, leaving everything else.
  // Uses router.replace so the back button doesn't trap the user in a loop.
  const clearPromptVersionFilter = useCallback(() => {
    const next = new URLSearchParams(searchParams.toString());
    next.delete('promptVersionId');
    const qs = next.toString();
    router.replace(`/dashboard/traces${qs ? '?' + qs : ''}`);
  }, [router, searchParams]);

  // Fetch traces when project ID or filters change, with polling
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
        const params = buildFilterParams();
        const result = await dashboardAPI.getTraces(projectId, params);
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
  }, [projectId, buildFilterParams]);

  // Get unique sessions from traces
  const allSessions = useMemo(() => {
    const sessionSet = new Set<string>();
    traces.forEach((trace) => {
      if (trace.sessionId) sessionSet.add(trace.sessionId);
    });
    return Array.from(sessionSet).sort();
  }, [traces]);

  // Get unique tags from traces
  const availableTags = useMemo(() => {
    const tagSet = new Set<string>();
    traces.forEach((trace) => {
      trace.tags?.forEach((tag: string) => tagSet.add(tag));
    });
    return Array.from(tagSet).sort();
  }, [traces]);

  // No more client-side filtering - backend handles it
  const filteredTraces = traces;

  // Clear selection if selected trace is no longer in list
  useEffect(() => {
    if (selectedTraceId && !traces.find(t => t.id === selectedTraceId)) {
      setSelectedTraceId(traces[0]?.id ?? null);
    }
  }, [traces, selectedTraceId]);

  // Handle date range change
  const handleDateRangeChange = (from: Date | null, to: Date | null) => {
    setDateFrom(from);
    setDateTo(to);
  };

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

  const hasActiveFilters = nameSearch || statusFilter !== 'all' || sessionFilter !== 'all' || selectedTags.length > 0 || dateFrom || dateTo;

  const clearAllFilters = () => {
    setNameSearch('');
    setStatusFilter('all');
    setSessionFilter('all');
    setSelectedTags([]);
    setDateFrom(null);
    setDateTo(null);
  };

  // When a trace is selected, use edge-to-edge layout
  if (selectedTraceId) {
    return (
      <div className="flex flex-col h-full overflow-hidden">
        {/* Master-Detail Layout - edge to edge */}
        <div className="flex-1 flex min-h-0">
          {/* Left Panel: Traces List */}
          <div className="w-72 flex-shrink-0 flex flex-col border-r border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 overflow-hidden">
            {/* Compact Filters */}
            <div className="flex-shrink-0 p-2 border-b border-zinc-200 dark:border-zinc-700 space-y-2 bg-zinc-50 dark:bg-zinc-800/50">
              <Input
                placeholder="Search by name..."
                value={nameSearch}
                onChange={(e) => setNameSearch(e.target.value)}
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
            <div className="flex-shrink-0 px-2 py-1.5 border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
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

          {/* Right Panel: Trace Detail - takes remaining space */}
          <div className="flex-1 bg-white dark:bg-zinc-900 overflow-hidden min-w-0">
            <TraceDetail
              traceId={selectedTraceId}
              onClose={() => setSelectedTraceId(null)}
            />
          </div>
        </div>
      </div>
    );
  }

  // When no trace is selected, show table with padding
  return (
    <div className="p-4 sm:p-6 lg:p-8 flex flex-col h-full overflow-hidden">
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

      {/* Table View */}
      <div className="flex-1 flex flex-col border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden">
        {/* Filters Row */}
        <div className="flex-shrink-0 p-4 border-b border-zinc-200 dark:border-zinc-700">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex-1 min-w-[200px] max-w-sm">
              <Input
                placeholder="Search by name..."
                value={nameSearch}
                onChange={(e) => setNameSearch(e.target.value)}
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
            <TagsFilter
              availableTags={availableTags}
              selectedTags={selectedTags}
              onChange={setSelectedTags}
            />
            <DateRangeFilter
              from={dateFrom}
              to={dateTo}
              onChange={handleDateRangeChange}
            />
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={clearAllFilters}
              >
                Clear filters
              </Button>
            )}
            <div className="ml-auto text-sm text-zinc-500 dark:text-zinc-400">
              {filteredTraces.length} traces
            </div>
          </div>

          {promptVersionIdFromUrl && (
            <div className="px-4 py-2 border-b border-zinc-200 dark:border-zinc-700 bg-amber-50 dark:bg-amber-500/10 flex items-center gap-2 text-xs">
              <span className="text-amber-700 dark:text-amber-300" aria-hidden>🧷</span>
              <span className="text-amber-800 dark:text-amber-200">
                Filtered to traces tagged with prompt version{' '}
                <code className="font-mono text-[11px] bg-amber-100 dark:bg-amber-500/20 px-1 py-0.5 rounded">
                  {promptVersionIdFromUrl.slice(0, 8)}…
                </code>
              </span>
              <button
                type="button"
                onClick={clearPromptVersionFilter}
                className="ml-auto text-amber-700 hover:text-amber-900 dark:text-amber-300 dark:hover:text-amber-100 underline underline-offset-2"
              >
                Clear
              </button>
            </div>
          )}
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
                      {trace.totalTokens > 0 ? trace.totalTokens.toLocaleString() : '-'}
                    </TableCell>
                    <TableCell className="text-right font-mono text-amber-600 dark:text-amber-400">
                      {trace.totalCostUsd > 0 ? `$${trace.totalCostUsd.toFixed(4)}` : '-'}
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
