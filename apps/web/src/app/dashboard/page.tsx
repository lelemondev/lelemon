'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useProject } from '@/lib/project-context';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

interface Stats {
  totalTraces: number;
  totalTokens: number;
  totalCostUsd: number;
  errorRate: number;
}

interface RecentTrace {
  id: string;
  sessionId: string | null;
  totalDurationMs: number;
  totalTokens: number;
  totalCostUsd: string;
  status: 'active' | 'completed' | 'error';
  createdAt: string;
}

function formatDuration(ms: number): string {
  if (ms === 0) return '-';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
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

export default function DashboardPage() {
  const { currentProject, isLoading: projectLoading } = useProject();
  const [stats, setStats] = useState<Stats | null>(null);
  const [recentTraces, setRecentTraces] = useState<RecentTrace[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!currentProject) {
      setStats(null);
      setRecentTraces([]);
      setIsLoading(false);
      return;
    }

    const fetchData = async () => {
      setIsLoading(true);
      try {
        const [statsRes, tracesRes] = await Promise.all([
          fetch(`/api/v1/dashboard/stats?projectId=${currentProject.id}`),
          fetch(`/api/v1/dashboard/traces?projectId=${currentProject.id}`),
        ]);

        if (statsRes.ok) {
          const statsData = await statsRes.json();
          setStats(statsData);
        }

        if (tracesRes.ok) {
          const tracesData = await tracesRes.json();
          setRecentTraces(tracesData.slice(0, 5));
        }
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [currentProject]);

  if (projectLoading) {
    return (
      <div className="space-y-8">
        <div className="h-12 w-64 bg-zinc-200 dark:bg-zinc-800 rounded animate-pulse" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-28 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
          ))}
        </div>
      </div>
    );
  }

  if (!currentProject) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <div className="w-20 h-20 rounded-full bg-amber-100 dark:bg-amber-500/10 flex items-center justify-center mb-6">
          <svg className="w-10 h-10 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
          </svg>
        </div>
        <h2 className="text-2xl font-bold text-zinc-900 dark:text-white mb-2">Welcome to Lelemon</h2>
        <p className="text-zinc-500 dark:text-zinc-400 mb-6 text-center max-w-md">
          Create your first project to start tracing your LLM applications.
        </p>
        <Link href="/dashboard/projects">
          <Button className="bg-amber-500 hover:bg-amber-600 text-zinc-900 font-medium">
            <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            Create Project
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Overview</h1>
        <p className="text-zinc-500 dark:text-zinc-400 mt-1">
          Monitor your LLM application performance at a glance.
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="p-6 bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
          <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400">Total Traces</p>
          {isLoading ? (
            <div className="h-9 w-24 bg-zinc-200 dark:bg-zinc-700 rounded mt-2 animate-pulse" />
          ) : (
            <p className="text-3xl font-bold text-zinc-900 dark:text-white mt-2">
              {stats?.totalTraces.toLocaleString() ?? '0'}
            </p>
          )}
        </div>

        <div className="p-6 bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
          <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400">Total Tokens</p>
          {isLoading ? (
            <div className="h-9 w-24 bg-zinc-200 dark:bg-zinc-700 rounded mt-2 animate-pulse" />
          ) : (
            <p className="text-3xl font-bold text-zinc-900 dark:text-white mt-2">
              {stats?.totalTokens ? `${(stats.totalTokens / 1000).toFixed(1)}k` : '0'}
            </p>
          )}
        </div>

        <div className="p-6 bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
          <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400">Total Cost</p>
          {isLoading ? (
            <div className="h-9 w-24 bg-zinc-200 dark:bg-zinc-700 rounded mt-2 animate-pulse" />
          ) : (
            <p className="text-3xl font-bold text-amber-600 dark:text-amber-400 mt-2">
              ${stats?.totalCostUsd?.toFixed(2) ?? '0.00'}
            </p>
          )}
        </div>

        <div className="p-6 bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
          <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400">Error Rate</p>
          {isLoading ? (
            <div className="h-9 w-24 bg-zinc-200 dark:bg-zinc-700 rounded mt-2 animate-pulse" />
          ) : (
            <p className={`text-3xl font-bold mt-2 ${
              (stats?.errorRate ?? 0) > 5
                ? 'text-red-600 dark:text-red-400'
                : 'text-emerald-600 dark:text-emerald-400'
            }`}>
              {stats?.errorRate?.toFixed(1) ?? '0'}%
            </p>
          )}
        </div>
      </div>

      {/* Recent Traces */}
      <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
        <div className="p-6 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">Recent Traces</h2>
          <Link href="/dashboard/traces">
            <Button variant="ghost" size="sm">
              View all
              <svg className="w-4 h-4 ml-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
              </svg>
            </Button>
          </Link>
        </div>
        {isLoading ? (
          <div className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="px-6 py-4">
                <div className="h-10 bg-zinc-200 dark:bg-zinc-700 rounded animate-pulse" />
              </div>
            ))}
          </div>
        ) : recentTraces.length === 0 ? (
          <div className="p-12 text-center">
            <p className="text-zinc-500 dark:text-zinc-400 mb-4">
              No traces yet. Start by sending traces from your application.
            </p>
            <div className="inline-flex items-center gap-2 px-4 py-2 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
              <code className="text-sm font-mono text-zinc-600 dark:text-zinc-300">
                LELEMON_API_KEY={currentProject.apiKey.slice(0, 12)}...
              </code>
            </div>
          </div>
        ) : (
          <div className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {recentTraces.map((trace) => (
              <Link
                key={trace.id}
                href={`/dashboard/traces/${trace.id}`}
                className="px-6 py-4 flex items-center justify-between hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors"
              >
                <div className="flex items-center gap-4">
                  <div className={`w-2 h-2 rounded-full ${
                    trace.status === 'completed' ? 'bg-emerald-500' :
                    trace.status === 'error' ? 'bg-red-500' : 'bg-amber-500'
                  }`} />
                  <div>
                    <p className="text-sm font-medium text-zinc-900 dark:text-white">
                      {formatRelativeTime(trace.createdAt)}
                    </p>
                    <p className="text-xs text-zinc-400 dark:text-zinc-500 font-mono">
                      {trace.sessionId?.slice(0, 16) || trace.id.slice(0, 8)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-8 text-sm text-zinc-500 dark:text-zinc-400">
                  <span className="w-16 text-right">{formatDuration(trace.totalDurationMs)}</span>
                  <span className="w-20 text-right">{trace.totalTokens.toLocaleString()} tok</span>
                  <span className="w-16 text-right font-mono text-amber-600 dark:text-amber-400">
                    ${parseFloat(trace.totalCostUsd).toFixed(3)}
                  </span>
                  <Badge
                    variant={
                      trace.status === 'completed'
                        ? 'default'
                        : trace.status === 'error'
                        ? 'destructive'
                        : 'secondary'
                    }
                    className="text-xs w-20 justify-center"
                  >
                    {trace.status}
                  </Badge>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
