'use client';

import { useEffect, useState, useCallback } from 'react';
import {
  Bar, BarChart, Area, AreaChart, Line, LineChart, Cell, Pie, PieChart,
  XAxis, YAxis, CartesianGrid, LabelList,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent,
  type ChartConfig,
} from '@/components/ui/chart';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useProject } from '@/lib/project-context';
import {
  dashboardAPI, Stats, UsageDataPoint, ModelStats, TagStats, UserStats,
  HourlyHeatmap, LatencyBucket, LatencyPoint, AnalyticsParams,
} from '@/lib/api';
import { resolveModelName } from '@/components/traces/display-context';
import { EnterpriseGate } from '@/ee/components/feature-gate';
import { CostBreakdownChart } from '@/ee/components/cost-breakdown-chart';
import { ErrorAnalytics } from '@/ee/components/error-analytics';

// Chart colors aligned with lemon design system
const CHART_COLORS = [
  'var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)',
  'var(--chart-4)', 'var(--chart-5)',
];

const tracesChartConfig = {
  traces: { label: 'Traces', color: 'var(--chart-1)' },
} satisfies ChartConfig;

const tokensChartConfig = {
  tokens: { label: 'Tokens', color: 'var(--chart-2)' },
} satisfies ChartConfig;

const costChartConfig = {
  costUsd: { label: 'Cost (USD)', color: 'var(--chart-3)' },
} satisfies ChartConfig;

const latencyChartConfig = {
  p50: { label: 'p50', color: 'var(--chart-2)' },
  p95: { label: 'p95', color: 'var(--chart-1)' },
  p99: { label: 'p99', color: 'var(--chart-3)' },
} satisfies ChartConfig;

const latencyDistConfig = {
  count: { label: 'Requests', color: 'var(--chart-1)' },
} satisfies ChartConfig;

// Date range presets
type DatePreset = '24h' | '7d' | '30d' | 'custom';

function getDateRange(preset: DatePreset): { from: string; to: string } {
  const to = new Date();
  const from = new Date();
  switch (preset) {
    case '24h': from.setHours(from.getHours() - 24); break;
    case '7d': from.setDate(from.getDate() - 7); break;
    case '30d': from.setDate(from.getDate() - 30); break;
    default: from.setDate(from.getDate() - 7);
  }
  return { from: from.toISOString(), to: to.toISOString() };
}

function formatDateDay(time: string): string {
  return new Date(time).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatDateHour(time: string): string {
  const d = new Date(time);
  return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
}

function formatDateFull(time: string): string {
  const d = new Date(time);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) +
    ' ' + d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
}

function formatTokens(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}k`;
  return String(value);
}

function formatDuration(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${ms}ms`;
}

function formatCost(usd: number): string {
  if (usd >= 1) return `$${usd.toFixed(2)}`;
  return `$${usd.toFixed(4)}`;
}

const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export default function AnalyticsPage() {
  const { currentProject, isLoading: projectLoading } = useProject();
  const [datePreset, setDatePreset] = useState<DatePreset>('7d');
  const [isLoading, setIsLoading] = useState(true); // first load only
  const [isRefreshing, setIsRefreshing] = useState(false); // subsequent loads
  const [error, setError] = useState<string | null>(null);

  // Filters (applied = sent to API, draft = in input fields)
  const [filterTag, setFilterTag] = useState('');
  const [filterSession, setFilterSession] = useState('');
  const [filterUser, setFilterUser] = useState('');
  const [filterName, setFilterName] = useState('');
  const [appliedFilters, setAppliedFilters] = useState({ tag: '', sessionId: '', userId: '', name: '' });
  const hasActiveFilters = !!(appliedFilters.tag || appliedFilters.sessionId || appliedFilters.userId || appliedFilters.name);

  const applyFilters = useCallback(() => {
    setAppliedFilters({ tag: filterTag, sessionId: filterSession, userId: filterUser, name: filterName });
  }, [filterTag, filterSession, filterUser, filterName]);

  const clearFilters = useCallback(() => {
    setFilterTag(''); setFilterSession(''); setFilterUser(''); setFilterName('');
    setAppliedFilters({ tag: '', sessionId: '', userId: '', name: '' });
  }, []);

  // Data
  const [stats, setStats] = useState<Stats | null>(null);
  const [usage, setUsage] = useState<UsageDataPoint[]>([]);
  const [models, setModels] = useState<ModelStats[]>([]);
  const [tags, setTags] = useState<TagStats[]>([]);
  const [topUsers, setTopUsers] = useState<UserStats[]>([]);
  const [heatmap, setHeatmap] = useState<HourlyHeatmap[]>([]);
  const [latencyDist, setLatencyDist] = useState<LatencyBucket[]>([]);
  const [latencyTS, setLatencyTS] = useState<LatencyPoint[]>([]);

  const hasData = stats !== null;

  const fetchAll = useCallback(async () => {
    if (!currentProject?.id) return;
    // Only show skeleton on first load; subsequent loads show subtle indicator
    if (hasData) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);

    const range = getDateRange(datePreset);
    const params: AnalyticsParams = {
      from: range.from,
      to: range.to,
      ...(appliedFilters.tag ? { tag: appliedFilters.tag } : {}),
      ...(appliedFilters.sessionId ? { sessionId: appliedFilters.sessionId } : {}),
      ...(appliedFilters.userId ? { userId: appliedFilters.userId } : {}),
      ...(appliedFilters.name ? { name: appliedFilters.name } : {}),
    };
    const granularity = datePreset === '24h' ? 'hour' : 'day';

    try {
      const results = await Promise.allSettled([
        dashboardAPI.getStats(currentProject.id, params),
        dashboardAPI.getUsage(currentProject.id, { ...params, granularity }),
        dashboardAPI.getModelStats(currentProject.id, params),
        dashboardAPI.getTagStats(currentProject.id, params),
        dashboardAPI.getTopUsers(currentProject.id, { ...params, limit: 10 }),
        dashboardAPI.getHeatmap(currentProject.id, params),
        dashboardAPI.getLatencyDistribution(currentProject.id, params),
        dashboardAPI.getLatencyTimeSeries(currentProject.id, { ...params, granularity }),
      ]);

      const val = <T,>(r: PromiseSettledResult<T>, fallback: T): T =>
        r.status === 'fulfilled' ? r.value : fallback;

      setStats(val(results[0], null));
      setUsage(val(results[1], []));
      setModels(val(results[2], []));
      setTags(val(results[3], []));
      setTopUsers(val(results[4], []));
      setHeatmap(val(results[5], []));
      setLatencyDist(val(results[6], []));
      setLatencyTS(val(results[7], []));

      const failures = results.filter((r) => r.status === 'rejected');
      if (failures.length === results.length) {
        setError('Failed to load analytics data');
      }
    } catch (err) {
      console.error('Failed to fetch analytics:', err);
      setError('Failed to load analytics data');
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, [currentProject?.id, datePreset, hasData, appliedFilters]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  const totalCost = stats?.totalCostUsd ?? 0;
  const totalTokens = stats?.totalTokens ?? 0;
  const totalTraces = stats?.totalTraces ?? 0;
  const avgDurationMs = stats?.avgDurationMs ?? 0;

  const isHourly = datePreset === '24h';
  const formatDate = isHourly ? formatDateHour : formatDateDay;
  const chartData = usage.map((day) => ({ ...day, date: formatDate(day.time), dateFull: formatDateFull(day.time) }));
  const latencyChartData = latencyTS.map((p) => ({
    ...p,
    date: formatDate(p.time),
  }));

  // Model aliases from project settings
  const modelAliases = ((currentProject?.settings as Record<string, unknown>)?.modelAliases ?? {}) as Record<string, string>;
  const displayModel = (model: string) => resolveModelName(model, modelAliases);

  // Model pie chart data
  const modelPieData = models.map((m, i) => ({
    name: displayModel(m.model),
    value: m.totalCostUsd,
    fill: CHART_COLORS[i % CHART_COLORS.length],
  }));
  const modelPieConfig = Object.fromEntries(
    modelPieData.map((m) => [m.name, { label: m.name, color: m.fill }])
  ) satisfies ChartConfig;

  if (projectLoading || isLoading) {
    return (
      <div className="p-4 sm:p-6 lg:p-8 space-y-6 overflow-auto h-full">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
          <p className="text-muted-foreground">Loading analytics...</p>
        </div>
        <div className="grid gap-4 md:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i}><CardHeader className="pb-2"><Skeleton className="h-4 w-24" /></CardHeader>
              <CardContent><Skeleton className="h-8 w-16" /></CardContent></Card>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 sm:p-6 lg:p-8 space-y-6 overflow-auto h-full">
        <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
        <p className="text-red-500">{error}</p>
      </div>
    );
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-6 overflow-auto h-full">
      {/* Header + Date Range Picker */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
          <p className="text-muted-foreground">Deep dive into your LLM usage and costs.</p>
        </div>
        <div className="flex items-center gap-2">
          {isRefreshing && (
            <div className="w-4 h-4 border-2 border-amber-500 border-t-transparent rounded-full animate-spin" />
          )}
          <div className="flex gap-1">
            {(['24h', '7d', '30d'] as DatePreset[]).map((preset) => (
              <Button
                key={preset}
                variant={datePreset === preset ? 'default' : 'outline'}
                size="sm"
                onClick={() => setDatePreset(preset)}
                disabled={isRefreshing}
              >
                {preset}
              </Button>
            ))}
            <Button
              variant="outline"
              size="sm"
              onClick={fetchAll}
              disabled={isRefreshing}
              title="Refresh data"
            >
              <svg className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" />
              </svg>
            </Button>
          </div>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="flex flex-wrap items-center gap-2">
        <Input
          placeholder="Tag (e.g. org:90)"
          value={filterTag}
          onChange={(e) => setFilterTag(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
          className="w-44 h-8 text-xs"
        />
        <Input
          placeholder="Session ID"
          value={filterSession}
          onChange={(e) => setFilterSession(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
          className="w-36 h-8 text-xs"
        />
        <Input
          placeholder="User ID"
          value={filterUser}
          onChange={(e) => setFilterUser(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
          className="w-36 h-8 text-xs"
        />
        <Input
          placeholder="Trace name"
          value={filterName}
          onChange={(e) => setFilterName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
          className="w-36 h-8 text-xs"
        />
        <Button
          size="sm"
          onClick={applyFilters}
          disabled={isRefreshing}
          className="h-8 text-xs bg-amber-500 hover:bg-amber-600 text-zinc-900 cursor-pointer"
        >
          Apply
        </Button>
        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="h-8 text-xs text-muted-foreground cursor-pointer"
          >
            Clear
          </Button>
        )}
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
          <TabsTrigger value="latency">Latency</TabsTrigger>
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="costs">Costs</TabsTrigger>
          <TabsTrigger value="errors">Errors</TabsTrigger>
        </TabsList>

        {/* ===== OVERVIEW TAB ===== */}
        <TabsContent value="overview" className="space-y-6 mt-6">
          <div className="grid gap-4 md:grid-cols-4">
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Total Traces</CardTitle></CardHeader>
              <CardContent><div className="text-2xl font-bold font-mono">{totalTraces.toLocaleString()}</div></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Total Tokens</CardTitle></CardHeader>
              <CardContent><div className="text-2xl font-bold font-mono">{formatTokens(totalTokens)}</div></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Total Cost</CardTitle></CardHeader>
              <CardContent><div className="text-2xl font-bold font-mono">${totalCost.toFixed(2)}</div></CardContent></Card>
            <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Avg Latency (p50)</CardTitle></CardHeader>
              <CardContent><div className="text-2xl font-bold font-mono">
                {models.length > 0 ? formatDuration(models.reduce((sum, m) => sum + m.p50LatencyMs * m.requests, 0) / Math.max(models.reduce((sum, m) => sum + m.requests, 0), 1)) : formatDuration(avgDurationMs)}
              </div></CardContent></Card>
          </div>

          {/* Token Usage Chart */}
          <Card>
            <CardHeader><CardTitle>{isHourly ? 'Hourly Token Usage' : 'Daily Token Usage'}</CardTitle></CardHeader>
            <CardContent className="pt-2">
              {chartData.length > 0 ? (
                <ChartContainer config={tokensChartConfig} className="h-72 w-full">
                  <BarChart data={chartData} accessibilityLayer margin={{ top: 20, right: 12, bottom: 0, left: 0 }}>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} width={56} tickFormatter={formatTokens} />
                    <ChartTooltip
                      content={({ active, payload }) => {
                        if (!active || !payload?.length) return null;
                        const d = payload[0].payload;
                        return (
                          <div className="rounded-lg border bg-background px-3 py-2 text-xs shadow-xl space-y-1">
                            <div className="font-medium">{d.dateFull}</div>
                            <div className="flex justify-between gap-4">
                              <span className="text-muted-foreground">Traces</span>
                              <span className="font-mono font-medium">{d.traces.toLocaleString()}</span>
                            </div>
                            <div className="flex justify-between gap-4">
                              <span className="text-muted-foreground">Spans</span>
                              <span className="font-mono font-medium">{d.spans.toLocaleString()}</span>
                            </div>
                            <div className="flex justify-between gap-4">
                              <span className="text-muted-foreground">Tokens</span>
                              <span className="font-mono font-medium">{formatTokens(d.tokens)}</span>
                            </div>
                            <div className="flex justify-between gap-4">
                              <span className="text-muted-foreground">Cost</span>
                              <span className="font-mono font-medium text-amber-500">${d.costUsd.toFixed(2)}</span>
                            </div>
                          </div>
                        );
                      }}
                    />
                    <Bar dataKey="tokens" fill="var(--color-tokens)" radius={[4, 4, 0, 0]}>
                      <LabelList
                        dataKey="tokens"
                        position="top"
                        className="fill-muted-foreground"
                        fontSize={10}
                        formatter={(v) => formatTokens(v as number)}
                      />
                    </Bar>
                  </BarChart>
                </ChartContainer>
              ) : (
                <div className="h-72 flex items-center justify-center text-muted-foreground">No data available.</div>
              )}
            </CardContent>
          </Card>

          {/* Top Models + Top Users side by side */}
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader><CardTitle>Top Models by Cost</CardTitle></CardHeader>
              <CardContent>
                {models.length > 0 ? (
                  <div className="font-mono text-sm space-y-2">
                    {models.slice(0, 5).map((m) => (
                      <div key={m.model} className="flex items-center justify-between">
                        <span className="truncate max-w-[200px]">{displayModel(m.model)}</span>
                        <span className="font-bold">{formatCost(m.totalCostUsd)}</span>
                      </div>
                    ))}
                  </div>
                ) : <p className="text-muted-foreground text-sm">No model data.</p>}
              </CardContent>
            </Card>
            <Card>
              <CardHeader><CardTitle>Top Users by Cost</CardTitle></CardHeader>
              <CardContent>
                {topUsers.length > 0 ? (
                  <div className="font-mono text-sm space-y-2">
                    {topUsers.slice(0, 5).map((u) => (
                      <div key={u.userId} className="flex items-center justify-between">
                        <span className="truncate max-w-[200px]">{u.userId}</span>
                        <span className="font-bold">{formatCost(u.totalCostUsd)}</span>
                      </div>
                    ))}
                  </div>
                ) : <p className="text-muted-foreground text-sm">No user data.</p>}
              </CardContent>
            </Card>
          </div>

          {/* Stats Summary */}
          <Card>
            <CardHeader><CardTitle>Statistics Summary</CardTitle></CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 font-mono text-sm">
                <div className="space-y-1"><p className="text-muted-foreground">Total Spans</p><p className="text-lg font-bold">{stats?.totalSpans?.toLocaleString() ?? 0}</p></div>
                <div className="space-y-1"><p className="text-muted-foreground">Error Rate</p><p className="text-lg font-bold">{(stats?.errorRate ?? 0).toFixed(1)}%</p></div>
                <div className="space-y-1"><p className="text-muted-foreground">Avg Cost/Trace</p><p className="text-lg font-bold">${totalTraces > 0 ? (totalCost / totalTraces).toFixed(4) : '0.0000'}</p></div>
                <div className="space-y-1"><p className="text-muted-foreground">Models Used</p><p className="text-lg font-bold">{models.length}</p></div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* ===== MODELS TAB ===== */}
        <TabsContent value="models" className="space-y-6 mt-6">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Cost Pie Chart */}
            <Card>
              <CardHeader><CardTitle>Cost by Model</CardTitle></CardHeader>
              <CardContent className="pt-2">
                {modelPieData.length > 0 ? (
                  <div>
                    <ChartContainer config={modelPieConfig} className="h-52 w-full">
                      <PieChart>
                        <Pie data={modelPieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={50} outerRadius={80}>
                          {modelPieData.map((entry, i) => (
                            <Cell key={i} fill={entry.fill} />
                          ))}
                        </Pie>
                        <ChartTooltip content={<ChartTooltipContent formatter={(value) => formatCost(value as number)} />} />
                      </PieChart>
                    </ChartContainer>
                    <div className="mt-3 space-y-1.5">
                      {modelPieData.map((m) => (
                        <div key={m.name} className="flex items-center justify-between text-sm">
                          <div className="flex items-center gap-2">
                            <div className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: m.fill }} />
                            <span className="text-muted-foreground truncate max-w-[180px]">{m.name}</span>
                          </div>
                          <span className="font-mono font-medium">{formatCost(m.value)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : <div className="h-72 flex items-center justify-center text-muted-foreground">No model data.</div>}
              </CardContent>
            </Card>

            {/* Latency by Model */}
            <Card>
              <CardHeader><CardTitle>Latency by Model</CardTitle><CardDescription>p50 / p95 / p99</CardDescription></CardHeader>
              <CardContent>
                {models.length > 0 ? (
                  <div className="font-mono text-sm space-y-3">
                    {models.map((m) => (
                      <div key={m.model} className="space-y-1">
                        <div className="flex justify-between">
                          <span className="truncate max-w-[180px] text-muted-foreground">{displayModel(m.model)}</span>
                          <span>{formatDuration(m.p50LatencyMs)} / {formatDuration(m.p95LatencyMs)} / {formatDuration(m.p99LatencyMs)}</span>
                        </div>
                        <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                          <div className="h-full bg-chart-1 rounded-full" style={{ width: `${Math.min((m.p50LatencyMs / Math.max(...models.map(x => x.p99LatencyMs), 1)) * 100, 100)}%` }} />
                        </div>
                      </div>
                    ))}
                  </div>
                ) : <p className="text-muted-foreground text-sm">No model data.</p>}
              </CardContent>
            </Card>
          </div>

          {/* Models Table */}
          <Card>
            <CardHeader><CardTitle>Model Breakdown</CardTitle></CardHeader>
            <CardContent>
              {models.length > 0 ? (
                <div className="font-mono text-sm overflow-x-auto">
                  <div className="grid grid-cols-7 gap-4 py-2 border-b font-medium min-w-[700px]">
                    <span>Model</span>
                    <span className="text-right">Requests</span>
                    <span className="text-right">Tokens</span>
                    <span className="text-right">Cost</span>
                    <span className="text-right">p50</span>
                    <span className="text-right">p95</span>
                    <span className="text-right">p99</span>
                  </div>
                  {models.map((m) => (
                    <div key={m.model} className="grid grid-cols-7 gap-4 py-2 border-b border-dashed last:border-0 min-w-[700px]">
                      <span className="truncate">{displayModel(m.model)}</span>
                      <span className="text-right">{m.requests.toLocaleString()}</span>
                      <span className="text-right">{formatTokens(m.totalTokens)}</span>
                      <span className="text-right">{formatCost(m.totalCostUsd)}</span>
                      <span className="text-right">{formatDuration(m.p50LatencyMs)}</span>
                      <span className="text-right">{formatDuration(m.p95LatencyMs)}</span>
                      <span className="text-right">{formatDuration(m.p99LatencyMs)}</span>
                    </div>
                  ))}
                </div>
              ) : <p className="text-muted-foreground text-sm text-center py-8">No model data available.</p>}
            </CardContent>
          </Card>

          {/* Tags Breakdown */}
          <Card>
            <CardHeader><CardTitle>Cost by Tag</CardTitle><CardDescription>Filter by prefix (e.g. org:) to see cost per organization</CardDescription></CardHeader>
            <CardContent>
              {tags.length > 0 ? (
                <div className="font-mono text-sm">
                  <div className="grid grid-cols-5 gap-4 py-2 border-b font-medium">
                    <span>Tag</span>
                    <span className="text-right">Traces</span>
                    <span className="text-right">Tokens</span>
                    <span className="text-right">Cost</span>
                    <span className="text-right">Avg Latency</span>
                  </div>
                  {tags.map((t) => (
                    <div key={t.tag} className="grid grid-cols-5 gap-4 py-2 border-b border-dashed last:border-0">
                      <span className="truncate">{t.tag}</span>
                      <span className="text-right">{t.traces.toLocaleString()}</span>
                      <span className="text-right">{formatTokens(t.totalTokens)}</span>
                      <span className="text-right">{formatCost(t.totalCostUsd)}</span>
                      <span className="text-right">{formatDuration(t.avgLatencyMs)}</span>
                    </div>
                  ))}
                </div>
              ) : <p className="text-muted-foreground text-sm text-center py-8">No tag data available.</p>}
            </CardContent>
          </Card>
        </TabsContent>

        {/* ===== LATENCY TAB ===== */}
        <TabsContent value="latency" className="space-y-6 mt-6">
          {/* p50/p95/p99 Cards */}
          {models.length > 0 && (
            <div className="grid gap-4 md:grid-cols-3">
              {(['p50LatencyMs', 'p95LatencyMs', 'p99LatencyMs'] as const).map((key) => {
                const label = key.replace('LatencyMs', '').toUpperCase();
                const weighted = models.reduce((sum, m) => sum + m[key] * m.requests, 0) / Math.max(models.reduce((sum, m) => sum + m.requests, 0), 1);
                return (
                  <Card key={key}><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">{label} Latency</CardTitle></CardHeader>
                    <CardContent><div className="text-2xl font-bold font-mono">{formatDuration(Math.round(weighted))}</div></CardContent></Card>
                );
              })}
            </div>
          )}

          {/* Latency Over Time */}
          <Card>
            <CardHeader><CardTitle>Latency Over Time</CardTitle><CardDescription>p50, p95, p99 percentiles</CardDescription></CardHeader>
            <CardContent className="pt-2">
              {latencyChartData.length > 0 ? (
                <ChartContainer config={latencyChartConfig} className="h-72 w-full">
                  <LineChart data={latencyChartData} accessibilityLayer margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} width={56} tickFormatter={(v: number) => formatDuration(v)} />
                    <ChartTooltip content={<ChartTooltipContent formatter={(value) => formatDuration(value as number)} />} />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Line type="monotone" dataKey="p50" stroke="var(--color-p50)" strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="p95" stroke="var(--color-p95)" strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="p99" stroke="var(--color-p99)" strokeWidth={2} dot={false} strokeDasharray="4 4" />
                  </LineChart>
                </ChartContainer>
              ) : <div className="h-72 flex items-center justify-center text-muted-foreground">No latency data.</div>}
            </CardContent>
          </Card>

          {/* Latency Distribution */}
          <Card>
            <CardHeader><CardTitle>Latency Distribution</CardTitle></CardHeader>
            <CardContent className="pt-2">
              {latencyDist.length > 0 ? (
                <ChartContainer config={latencyDistConfig} className="h-64 w-full">
                  <BarChart data={latencyDist} accessibilityLayer margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="bucket" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} width={48} />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Bar dataKey="count" fill="var(--color-count)" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ChartContainer>
              ) : <div className="h-64 flex items-center justify-center text-muted-foreground">No latency data.</div>}
            </CardContent>
          </Card>
        </TabsContent>

        {/* ===== USAGE TAB ===== */}
        <TabsContent value="usage" className="space-y-6 mt-6">
          {/* Token Usage Area Chart */}
          <Card>
            <CardHeader><CardTitle>Token Usage Over Time</CardTitle></CardHeader>
            <CardContent className="pt-2">
              {chartData.length > 0 ? (
                <ChartContainer config={tokensChartConfig} className="h-72 w-full">
                  <AreaChart data={chartData} accessibilityLayer margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} width={56} tickFormatter={formatTokens} />
                    <ChartTooltip content={<ChartTooltipContent formatter={(value) => formatTokens(value as number)} />} />
                    <defs><linearGradient id="fillTokens" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="var(--color-tokens)" stopOpacity={0.8} /><stop offset="95%" stopColor="var(--color-tokens)" stopOpacity={0.1} /></linearGradient></defs>
                    <Area dataKey="tokens" type="monotone" fill="url(#fillTokens)" stroke="var(--color-tokens)" strokeWidth={2} />
                  </AreaChart>
                </ChartContainer>
              ) : <div className="h-72 flex items-center justify-center text-muted-foreground">No usage data.</div>}
            </CardContent>
          </Card>

          {/* Heatmap */}
          <Card>
            <CardHeader><CardTitle>Activity Heatmap</CardTitle><CardDescription>Traces by hour and day of week</CardDescription></CardHeader>
            <CardContent>
              {heatmap.length > 0 ? (
                <div className="overflow-x-auto">
                  <div className="grid gap-0.5 min-w-[600px]" style={{ gridTemplateColumns: 'auto repeat(24, 1fr)' }}>
                    <div />
                    {Array.from({ length: 24 }, (_, i) => (
                      <div key={i} className="text-center text-xs text-muted-foreground font-mono">{i}</div>
                    ))}
                    {DAY_NAMES.map((day, dayIdx) => {
                      const maxTraces = Math.max(...heatmap.map((h) => h.traces), 1);
                      return (
                        <div key={`row-${dayIdx}`} className="contents">
                          <div className="text-xs text-muted-foreground font-mono pr-2 flex items-center">{day}</div>
                          {Array.from({ length: 24 }, (_, hour) => {
                            const cell = heatmap.find((h) => h.day === dayIdx && h.hour === hour);
                            const intensity = cell ? cell.traces / maxTraces : 0;
                            const opacity = Math.max(intensity, 0.05);
                            return (
                              <div
                                key={`${dayIdx}-${hour}`}
                                className="aspect-square rounded-sm"
                                style={{ backgroundColor: `var(--chart-1)`, opacity }}
                                title={cell ? `${day} ${hour}:00 — ${cell.traces} traces, ${formatCost(cell.costUsd)}` : `${day} ${hour}:00 — 0 traces`}
                              />
                            );
                          })}
                        </div>
                      );
                    })}
                  </div>
                </div>
              ) : <p className="text-muted-foreground text-sm text-center py-8">No heatmap data.</p>}
            </CardContent>
          </Card>

          {/* Daily Breakdown Table */}
          <Card>
            <CardHeader><CardTitle>Daily Breakdown</CardTitle></CardHeader>
            <CardContent>
              {usage.length > 0 ? (
                <div className="font-mono text-sm">
                  <div className="grid grid-cols-5 gap-4 py-2 border-b font-medium">
                    <span>Date</span><span className="text-right">Traces</span><span className="text-right">Spans</span><span className="text-right">Tokens</span><span className="text-right">Cost</span>
                  </div>
                  {usage.map((day) => (
                    <div key={day.time} className="grid grid-cols-5 gap-4 py-2 border-b border-dashed last:border-0">
                      <span>{formatDate(day.time)}</span>
                      <span className="text-right">{day.traces}</span>
                      <span className="text-right">{day.spans}</span>
                      <span className="text-right">{formatTokens(day.tokens)}</span>
                      <span className="text-right">${day.costUsd.toFixed(2)}</span>
                    </div>
                  ))}
                </div>
              ) : <p className="text-muted-foreground text-sm text-center py-8">No usage data.</p>}
            </CardContent>
          </Card>

          {/* Top Users */}
          <Card>
            <CardHeader><CardTitle>Top Users</CardTitle></CardHeader>
            <CardContent>
              {topUsers.length > 0 ? (
                <div className="font-mono text-sm">
                  <div className="grid grid-cols-5 gap-4 py-2 border-b font-medium">
                    <span>User</span><span className="text-right">Traces</span><span className="text-right">Tokens</span><span className="text-right">Cost</span><span className="text-right">Avg Latency</span>
                  </div>
                  {topUsers.map((u) => (
                    <div key={u.userId} className="grid grid-cols-5 gap-4 py-2 border-b border-dashed last:border-0">
                      <span className="truncate">{u.userId}</span>
                      <span className="text-right">{u.traces}</span>
                      <span className="text-right">{formatTokens(u.totalTokens)}</span>
                      <span className="text-right">{formatCost(u.totalCostUsd)}</span>
                      <span className="text-right">{formatDuration(u.avgLatencyMs)}</span>
                    </div>
                  ))}
                </div>
              ) : <p className="text-muted-foreground text-sm text-center py-8">No user data.</p>}
            </CardContent>
          </Card>
        </TabsContent>

        {/* ===== COSTS TAB ===== */}
        <TabsContent value="costs" className="space-y-6 mt-6">
          <EnterpriseGate
            fallback={
              <div className="space-y-6">
                <Card>
                  <CardHeader><CardTitle>Daily Cost</CardTitle></CardHeader>
                  <CardContent className="pt-2">
                    {chartData.length > 0 ? (
                      <ChartContainer config={costChartConfig} className="h-72 w-full">
                        <AreaChart data={chartData} accessibilityLayer margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
                          <CartesianGrid vertical={false} />
                          <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                          <YAxis tickLine={false} axisLine={false} tickMargin={8} width={56} tickFormatter={(value: number) => `$${value.toFixed(0)}`} />
                          <ChartTooltip content={<ChartTooltipContent formatter={(value) => `$${(value as number).toFixed(2)}`} />} />
                          <defs><linearGradient id="fillCost" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="var(--color-costUsd)" stopOpacity={0.8} /><stop offset="95%" stopColor="var(--color-costUsd)" stopOpacity={0.1} /></linearGradient></defs>
                          <Area dataKey="costUsd" type="monotone" fill="url(#fillCost)" stroke="var(--color-costUsd)" strokeWidth={2} />
                        </AreaChart>
                      </ChartContainer>
                    ) : <div className="h-72 flex items-center justify-center text-muted-foreground">No cost data.</div>}
                  </CardContent>
                </Card>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Total Cost</CardTitle></CardHeader>
                    <CardContent><div className="text-2xl font-bold font-mono">${totalCost.toFixed(2)}</div></CardContent></Card>
                  <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Total Tokens</CardTitle></CardHeader>
                    <CardContent><div className="text-2xl font-bold font-mono">{formatTokens(totalTokens)}</div></CardContent></Card>
                  <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Cost per 1k Tokens</CardTitle></CardHeader>
                    <CardContent><div className="text-2xl font-bold font-mono">${totalTokens > 0 ? ((totalCost / totalTokens) * 1000).toFixed(4) : '0.0000'}</div></CardContent></Card>
                  <Card><CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">Avg Cost/Trace</CardTitle></CardHeader>
                    <CardContent><div className="text-2xl font-bold font-mono">${totalTraces > 0 ? (totalCost / totalTraces).toFixed(4) : '0.0000'}</div></CardContent></Card>
                </div>
              </div>
            }
          >
            {currentProject && <CostBreakdownChart projectId={currentProject.id} />}
          </EnterpriseGate>
        </TabsContent>

        {/* ===== ERRORS TAB ===== */}
        <TabsContent value="errors" className="space-y-6 mt-6">
          <EnterpriseGate
            fallback={
              <Card><CardHeader><CardTitle>Error Analytics</CardTitle></CardHeader>
                <CardContent>
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <h3 className="text-lg font-semibold mb-2">Enterprise Feature</h3>
                    <p className="text-muted-foreground max-w-md">Error analytics with tag-based breakdown is available in the Enterprise edition.</p>
                  </div>
                </CardContent>
              </Card>
            }
          >
            {currentProject && <ErrorAnalytics projectId={currentProject.id} />}
          </EnterpriseGate>
        </TabsContent>
      </Tabs>
    </div>
  );
}
