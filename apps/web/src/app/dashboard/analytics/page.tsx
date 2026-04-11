'use client';

import { useEffect, useState, useCallback } from 'react';
import { Bar, BarChart, Area, AreaChart, XAxis, YAxis, CartesianGrid } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Skeleton } from '@/components/ui/skeleton';
import { useProject } from '@/lib/project-context';
import { dashboardAPI, Stats, UsageDataPoint } from '@/lib/api';
import { EnterpriseGate } from '@/ee/components/feature-gate';
import { CostBreakdownChart } from '@/ee/components/cost-breakdown-chart';
import { ErrorAnalytics } from '@/ee/components/error-analytics';

const tracesChartConfig = {
  traces: { label: 'Traces', color: 'var(--chart-1)' },
} satisfies ChartConfig;

const tokensChartConfig = {
  tokens: { label: 'Tokens', color: 'var(--chart-2)' },
} satisfies ChartConfig;

const costChartConfig = {
  costUsd: { label: 'Cost (USD)', color: 'var(--chart-3)' },
} satisfies ChartConfig;

function formatDate(time: string): string {
  return new Date(time).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatTokens(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}k`;
  return String(value);
}

export default function AnalyticsPage() {
  const { currentProject, isLoading: projectLoading } = useProject();
  const [stats, setStats] = useState<Stats | null>(null);
  const [usage, setUsage] = useState<UsageDataPoint[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAnalytics = useCallback(async () => {
    if (!currentProject?.id) return;

    setIsLoading(true);
    setError(null);

    try {
      const [statsData, usageData] = await Promise.all([
        dashboardAPI.getStats(currentProject.id),
        dashboardAPI.getUsage(currentProject.id, { granularity: 'day' }),
      ]);
      setStats(statsData);
      setUsage(usageData);
    } catch (err) {
      console.error('Failed to fetch analytics:', err);
      setError('Failed to load analytics data');
    } finally {
      setIsLoading(false);
    }
  }, [currentProject?.id]);

  useEffect(() => {
    fetchAnalytics();
  }, [fetchAnalytics]);

  const totalCost = stats?.totalCostUsd ?? 0;
  const totalTokens = stats?.totalTokens ?? 0;
  const totalTraces = stats?.totalTraces ?? 0;
  const avgDurationMs = stats?.avgDurationMs ?? 0;

  const chartData = usage.map((day) => ({
    ...day,
    date: formatDate(day.time),
  }));

  if (projectLoading || isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
          <p className="text-muted-foreground">Loading analytics...</p>
        </div>
        <div className="grid gap-4 md:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i}>
              <CardHeader className="pb-2">
                <Skeleton className="h-4 w-24" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-16" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
          <p className="text-red-500">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
        <p className="text-muted-foreground">
          Deep dive into your LLM usage and costs.
        </p>
      </div>

      <Tabs defaultValue="overview">
        <TabsList className="grid w-full max-w-lg grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="costs">Costs</TabsTrigger>
          <TabsTrigger value="errors">Errors</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6 mt-6">
          {/* Summary Cards */}
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Total Traces (7d)
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold font-mono">
                  {totalTraces.toLocaleString()}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Total Tokens (7d)
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold font-mono">
                  {formatTokens(totalTokens)}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Total Cost (7d)
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold font-mono">
                  ${totalCost.toFixed(2)}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Avg Duration
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold font-mono">
                  {avgDurationMs > 1000
                    ? `${(avgDurationMs / 1000).toFixed(1)}s`
                    : `${avgDurationMs}ms`}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Daily Traces Bar Chart */}
          <Card>
            <CardHeader>
              <CardTitle>Daily Traces</CardTitle>
            </CardHeader>
            <CardContent>
              {chartData.length > 0 ? (
                <ChartContainer config={tracesChartConfig} className="h-64 w-full">
                  <BarChart data={chartData} accessibilityLayer>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} allowDecimals={false} />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Bar dataKey="traces" fill="var(--color-traces)" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ChartContainer>
              ) : (
                <div className="h-64 flex items-center justify-center text-muted-foreground">
                  No usage data available for the selected period.
                </div>
              )}
            </CardContent>
          </Card>

          {/* Stats Summary */}
          <Card>
            <CardHeader>
              <CardTitle>Statistics Summary</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 font-mono text-sm">
                <div className="space-y-1">
                  <p className="text-muted-foreground">Total Spans</p>
                  <p className="text-lg font-bold">{stats?.totalSpans?.toLocaleString() ?? 0}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-muted-foreground">Avg Duration</p>
                  <p className="text-lg font-bold">
                    {avgDurationMs > 1000
                      ? `${(avgDurationMs / 1000).toFixed(1)}s`
                      : `${avgDurationMs}ms`}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-muted-foreground">Error Rate</p>
                  <p className="text-lg font-bold">{(stats?.errorRate ?? 0).toFixed(1)}%</p>
                </div>
                <div className="space-y-1">
                  <p className="text-muted-foreground">Avg Cost/Trace</p>
                  <p className="text-lg font-bold">
                    ${totalTraces > 0 ? (totalCost / totalTraces).toFixed(4) : '0.0000'}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="usage" className="space-y-6 mt-6">
          {/* Token Usage Area Chart */}
          <Card>
            <CardHeader>
              <CardTitle>Token Usage Over Time</CardTitle>
            </CardHeader>
            <CardContent>
              {chartData.length > 0 ? (
                <ChartContainer config={tokensChartConfig} className="h-64 w-full">
                  <AreaChart data={chartData} accessibilityLayer>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} tickFormatter={formatTokens} />
                    <ChartTooltip
                      content={
                        <ChartTooltipContent
                          formatter={(value) => formatTokens(value as number)}
                        />
                      }
                    />
                    <defs>
                      <linearGradient id="fillTokens" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="var(--color-tokens)" stopOpacity={0.8} />
                        <stop offset="95%" stopColor="var(--color-tokens)" stopOpacity={0.1} />
                      </linearGradient>
                    </defs>
                    <Area
                      dataKey="tokens"
                      type="monotone"
                      fill="url(#fillTokens)"
                      stroke="var(--color-tokens)"
                      strokeWidth={2}
                    />
                  </AreaChart>
                </ChartContainer>
              ) : (
                <div className="h-64 flex items-center justify-center text-muted-foreground">
                  No usage data available for the selected period.
                </div>
              )}
            </CardContent>
          </Card>

          {/* Usage Details Table */}
          <Card>
            <CardHeader>
              <CardTitle>Daily Breakdown</CardTitle>
            </CardHeader>
            <CardContent>
              {usage.length > 0 ? (
                <div className="font-mono text-sm">
                  <div className="grid grid-cols-5 gap-4 py-2 border-b font-medium">
                    <span>Date</span>
                    <span className="text-right">Traces</span>
                    <span className="text-right">Spans</span>
                    <span className="text-right">Tokens</span>
                    <span className="text-right">Cost</span>
                  </div>
                  {usage.map((day) => (
                    <div
                      key={day.time}
                      className="grid grid-cols-5 gap-4 py-2 border-b border-dashed last:border-0"
                    >
                      <span>{formatDate(day.time)}</span>
                      <span className="text-right">{day.traces}</span>
                      <span className="text-right">{day.spans}</span>
                      <span className="text-right">{formatTokens(day.tokens)}</span>
                      <span className="text-right">${day.costUsd.toFixed(2)}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No usage data available.
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="costs" className="space-y-6 mt-6">
          {/* Enterprise: Cost Breakdown by Tags */}
          <EnterpriseGate
            fallback={
              <div className="space-y-6">
                {/* Cost Area Chart */}
                <Card>
                  <CardHeader>
                    <CardTitle>Daily Cost</CardTitle>
                  </CardHeader>
                  <CardContent>
                    {chartData.length > 0 ? (
                      <ChartContainer config={costChartConfig} className="h-64 w-full">
                        <AreaChart data={chartData} accessibilityLayer>
                          <CartesianGrid vertical={false} />
                          <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                          <YAxis
                            tickLine={false}
                            axisLine={false}
                            tickMargin={8}
                            tickFormatter={(value: number) => `$${value.toFixed(2)}`}
                          />
                          <ChartTooltip
                            content={
                              <ChartTooltipContent
                                formatter={(value) => `$${(value as number).toFixed(4)}`}
                              />
                            }
                          />
                          <defs>
                            <linearGradient id="fillCost" x1="0" y1="0" x2="0" y2="1">
                              <stop offset="5%" stopColor="var(--color-costUsd)" stopOpacity={0.8} />
                              <stop offset="95%" stopColor="var(--color-costUsd)" stopOpacity={0.1} />
                            </linearGradient>
                          </defs>
                          <Area
                            dataKey="costUsd"
                            type="monotone"
                            fill="url(#fillCost)"
                            stroke="var(--color-costUsd)"
                            strokeWidth={2}
                          />
                        </AreaChart>
                      </ChartContainer>
                    ) : (
                      <div className="h-64 flex items-center justify-center text-muted-foreground">
                        No cost data available for the selected period.
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Cost Summary Cards */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Total Cost (7d)
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold font-mono">${totalCost.toFixed(2)}</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Total Tokens
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold font-mono">{formatTokens(totalTokens)}</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Cost per 1k Tokens
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold font-mono">
                        ${totalTokens > 0 ? ((totalCost / totalTokens) * 1000).toFixed(4) : '0.0000'}
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Avg Cost/Trace
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold font-mono">
                        ${totalTraces > 0 ? (totalCost / totalTraces).toFixed(4) : '0.0000'}
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </div>
            }
          >
            {currentProject && (
              <CostBreakdownChart projectId={currentProject.id} />
            )}
          </EnterpriseGate>
        </TabsContent>

        <TabsContent value="errors" className="space-y-6 mt-6">
          {/* Enterprise: Error Analytics */}
          <EnterpriseGate
            fallback={
              <Card>
                <CardHeader>
                  <CardTitle>Error Analytics</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <div className="text-4xl mb-4">🔒</div>
                    <h3 className="text-lg font-semibold mb-2">Enterprise Feature</h3>
                    <p className="text-muted-foreground max-w-md">
                      Error analytics with tag-based breakdown and top errors tracking
                      is available in the Enterprise edition.
                    </p>
                  </div>
                </CardContent>
              </Card>
            }
          >
            {currentProject && (
              <ErrorAnalytics projectId={currentProject.id} />
            )}
          </EnterpriseGate>
        </TabsContent>
      </Tabs>
    </div>
  );
}
