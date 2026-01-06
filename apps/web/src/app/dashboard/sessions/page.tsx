'use client';

import { Suspense, useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { useProject } from '@/lib/project-context';
import { dashboardAPI, Session } from '@/lib/api';
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
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function SessionsPageContent() {
  const { currentProject } = useProject();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [search, setSearch] = useState('');

  // Polling interval in ms
  const POLLING_INTERVAL = 5000;

  // Fetch sessions when project ID changes, with polling
  const projectId = currentProject?.id;
  useEffect(() => {
    if (!projectId) {
      setSessions([]);
      setIsLoading(false);
      return;
    }

    let isMounted = true;

    const fetchSessions = async (showLoading = false) => {
      if (showLoading) setIsLoading(true);
      try {
        const result = await dashboardAPI.getSessions(projectId, { limit: 100 });
        if (isMounted) {
          setSessions(result.data);
        }
      } catch (error) {
        console.error('Failed to fetch sessions:', error);
      } finally {
        if (isMounted && showLoading) {
          setIsLoading(false);
        }
      }
    };

    // Initial fetch with loading state
    fetchSessions(true);

    // Set up polling
    const intervalId = setInterval(() => {
      fetchSessions(false);
    }, POLLING_INTERVAL);

    // Cleanup on unmount
    return () => {
      isMounted = false;
      clearInterval(intervalId);
    };
  }, [projectId]);

  // Filter sessions by search
  const filteredSessions = useMemo(() => {
    return sessions.filter((session) => {
      if (!search) return true;
      const searchLower = search.toLowerCase();
      return (
        session.sessionId.toLowerCase().includes(searchLower) ||
        session.userId?.toLowerCase().includes(searchLower)
      );
    });
  }, [sessions, search]);

  if (!currentProject) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-500/10 flex items-center justify-center mb-4">
          <svg className="w-8 h-8 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">No project selected</h3>
        <p className="text-zinc-500 dark:text-zinc-400 mb-4">Select or create a project to view sessions.</p>
        <Link href="/dashboard/projects">
          <Button className="bg-amber-500 hover:bg-amber-600 text-zinc-900">
            Go to Projects
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-6 overflow-auto h-full">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Sessions</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            View conversations grouped by session. Each session contains multiple traces (turns).
          </p>
        </div>
        <div className="text-sm text-zinc-500 dark:text-zinc-400">
          {filteredSessions.length} sessions
        </div>
      </div>

      {/* Search */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <div className="flex-1 max-w-sm">
              <Input
                placeholder="Search by session ID or user..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full"
              />
            </div>
            {search && (
              <Button variant="ghost" size="sm" onClick={() => setSearch('')}>
                Clear
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Sessions Table */}
      <Card>
        <CardHeader>
          <CardTitle>All Sessions</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="h-12 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse" />
              ))}
            </div>
          ) : filteredSessions.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-zinc-500 dark:text-zinc-400">
                {sessions.length === 0
                  ? 'No sessions yet. Sessions are created when traces include a sessionId.'
                  : 'No sessions match your search.'}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Session ID</TableHead>
                  <TableHead className="hidden md:table-cell">User</TableHead>
                  <TableHead className="text-right">Turns</TableHead>
                  <TableHead className="hidden sm:table-cell text-right">Spans</TableHead>
                  <TableHead className="text-right">Tokens</TableHead>
                  <TableHead className="text-right">Cost</TableHead>
                  <TableHead className="hidden lg:table-cell text-right">Duration</TableHead>
                  <TableHead className="text-center">Status</TableHead>
                  <TableHead className="hidden md:table-cell">Last Activity</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSessions.map((session) => (
                  <TableRow key={session.sessionId} className="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                    <TableCell>
                      <Link
                        href={`/dashboard/traces?sessionId=${encodeURIComponent(session.sessionId)}`}
                        className="font-mono text-sm text-amber-600 dark:text-amber-400 hover:underline"
                      >
                        {session.sessionId.length > 20
                          ? `${session.sessionId.slice(0, 20)}...`
                          : session.sessionId}
                      </Link>
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-zinc-600 dark:text-zinc-400">
                      {session.userId || 'Anonymous'}
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="inline-flex items-center gap-1 text-zinc-900 dark:text-white font-medium">
                        {session.traceCount}
                        <svg className="w-3.5 h-3.5 text-zinc-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
                        </svg>
                      </span>
                    </TableCell>
                    <TableCell className="hidden sm:table-cell text-right text-zinc-600 dark:text-zinc-400">
                      {session.totalSpans}
                    </TableCell>
                    <TableCell className="text-right text-zinc-600 dark:text-zinc-400">
                      {session.totalTokens > 0 ? session.totalTokens.toLocaleString() : '-'}
                    </TableCell>
                    <TableCell className="text-right font-mono text-amber-600 dark:text-amber-400">
                      {session.totalCostUsd > 0 ? `$${session.totalCostUsd.toFixed(4)}` : '-'}
                    </TableCell>
                    <TableCell className="hidden lg:table-cell text-right text-zinc-600 dark:text-zinc-400">
                      {formatDuration(session.totalDurationMs)}
                    </TableCell>
                    <TableCell className="text-center">
                      {session.hasActive ? (
                        <Badge variant="secondary" className="text-xs">
                          <span className="relative flex h-2 w-2 mr-1.5">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                          </span>
                          active
                        </Badge>
                      ) : session.hasError ? (
                        <Badge variant="destructive" className="text-xs">
                          <svg className="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                          </svg>
                          error
                        </Badge>
                      ) : (
                        <Badge variant="default" className="text-xs">
                          <svg className="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                          </svg>
                          completed
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-zinc-500 dark:text-zinc-400">
                      {formatRelativeTime(session.lastTraceAt)}
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

function SessionsPageFallback() {
  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Sessions</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            View conversations grouped by session.
          </p>
        </div>
      </div>
      <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
    </div>
  );
}

export default function SessionsPage() {
  return (
    <Suspense fallback={<SessionsPageFallback />}>
      <SessionsPageContent />
    </Suspense>
  );
}
