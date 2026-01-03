'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

// Mock analytics data
const dailyUsage = [
  { date: '2024-01-01', traces: 145, tokens: 45890, cost: 1.23 },
  { date: '2024-01-02', traces: 178, tokens: 52340, cost: 1.56 },
  { date: '2024-01-03', traces: 156, tokens: 48230, cost: 1.34 },
  { date: '2024-01-04', traces: 189, tokens: 61200, cost: 1.78 },
  { date: '2024-01-05', traces: 201, tokens: 68450, cost: 1.92 },
  { date: '2024-01-06', traces: 167, tokens: 51230, cost: 1.45 },
  { date: '2024-01-07', traces: 143, tokens: 42100, cost: 1.18 },
];

const modelBreakdown = [
  { model: 'gpt-4-turbo', traces: 523, tokens: 234567, cost: 5.67, pct: 45 },
  { model: 'gpt-4o', traces: 312, tokens: 145234, cost: 2.34, pct: 27 },
  { model: 'claude-3-5-sonnet', traces: 189, tokens: 89234, cost: 1.45, pct: 16 },
  { model: 'gpt-4o-mini', traces: 143, tokens: 78234, cost: 0.23, pct: 12 },
];

export default function AnalyticsPage() {
  const totalCost = dailyUsage.reduce((sum, d) => sum + d.cost, 0);
  const totalTokens = dailyUsage.reduce((sum, d) => sum + d.tokens, 0);
  const totalTraces = dailyUsage.reduce((sum, d) => sum + d.traces, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
        <p className="text-muted-foreground">
          Deep dive into your LLM usage and costs.
        </p>
      </div>

      <Tabs defaultValue="overview">
        <TabsList className="grid w-full max-w-md grid-cols-3">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="costs">Costs</TabsTrigger>
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
                  {(totalTokens / 1000).toFixed(0)}k
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
                  Avg Cost/Trace
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold font-mono">
                  ${(totalCost / totalTraces).toFixed(4)}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Daily Usage Chart (ASCII style for dev feel) */}
          <Card>
            <CardHeader>
              <CardTitle>Daily Usage</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="font-mono text-sm space-y-1">
                {dailyUsage.map((day) => {
                  const barLength = Math.round((day.traces / 210) * 40);
                  return (
                    <div key={day.date} className="flex items-center gap-2">
                      <span className="w-24 text-muted-foreground">
                        {new Date(day.date).toLocaleDateString('en-US', {
                          weekday: 'short',
                          month: 'short',
                          day: 'numeric',
                        })}
                      </span>
                      <div className="flex-1 flex items-center gap-2">
                        <div className="text-lime-500 dark:text-lime-400">
                          {'█'.repeat(barLength)}
                          {'░'.repeat(40 - barLength)}
                        </div>
                        <span className="w-16 text-right">{day.traces}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>

          {/* Model Breakdown */}
          <Card>
            <CardHeader>
              <CardTitle>Model Breakdown</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {modelBreakdown.map((model) => (
                  <div key={model.model} className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-mono">{model.model}</span>
                      <span className="text-muted-foreground">
                        {model.traces} traces · {(model.tokens / 1000).toFixed(0)}k tokens · ${model.cost.toFixed(2)}
                      </span>
                    </div>
                    <div className="h-2 bg-muted rounded-full overflow-hidden">
                      <div
                        className="h-full bg-lime-500 dark:bg-lime-400 rounded-full"
                        style={{ width: `${model.pct}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="usage" className="space-y-6 mt-6">
          <Card>
            <CardHeader>
              <CardTitle>Token Usage Over Time</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="h-64 flex items-end justify-between gap-1">
                {dailyUsage.map((day) => {
                  const height = (day.tokens / 70000) * 100;
                  return (
                    <div
                      key={day.date}
                      className="flex-1 flex flex-col items-center gap-1"
                    >
                      <div
                        className="w-full bg-lime-500/80 dark:bg-lime-400/80 rounded-t"
                        style={{ height: `${height}%` }}
                      />
                      <span className="text-xs text-muted-foreground font-mono">
                        {new Date(day.date).getDate()}
                      </span>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="costs" className="space-y-6 mt-6">
          <Card>
            <CardHeader>
              <CardTitle>Cost Breakdown</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="font-mono text-sm">
                <div className="grid grid-cols-5 gap-4 py-2 border-b font-medium">
                  <span>Model</span>
                  <span className="text-right">Input $</span>
                  <span className="text-right">Output $</span>
                  <span className="text-right">Total $</span>
                  <span className="text-right">% of Total</span>
                </div>
                {modelBreakdown.map((model) => (
                  <div
                    key={model.model}
                    className="grid grid-cols-5 gap-4 py-2 border-b border-dashed last:border-0"
                  >
                    <span>{model.model}</span>
                    <span className="text-right text-muted-foreground">
                      ${(model.cost * 0.4).toFixed(2)}
                    </span>
                    <span className="text-right text-muted-foreground">
                      ${(model.cost * 0.6).toFixed(2)}
                    </span>
                    <span className="text-right">${model.cost.toFixed(2)}</span>
                    <span className="text-right">{model.pct}%</span>
                  </div>
                ))}
                <div className="grid grid-cols-5 gap-4 py-2 font-bold">
                  <span>TOTAL</span>
                  <span className="text-right">
                    ${(totalCost * 0.4).toFixed(2)}
                  </span>
                  <span className="text-right">
                    ${(totalCost * 0.6).toFixed(2)}
                  </span>
                  <span className="text-right">${totalCost.toFixed(2)}</span>
                  <span className="text-right">100%</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
