'use client';

import { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { dashboardAPI, CostBreakdownResult, CostBreakdownFilter } from '@/lib/api';

interface CostBreakdownChartProps {
  projectId: string;
}

export function CostBreakdownChart({ projectId }: CostBreakdownChartProps) {
  const [data, setData] = useState<CostBreakdownResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tagPrefix, setTagPrefix] = useState('');

  const fetchData = useCallback(async (filter?: CostBreakdownFilter) => {
    try {
      setLoading(true);
      setError(null);
      const result = await dashboardAPI.getCostBreakdown(projectId, filter);
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load cost breakdown');
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleFilter = () => {
    fetchData({ tagPrefix: tagPrefix || undefined });
  };

  const formatCost = (cost: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 4,
      maximumFractionDigits: 4,
    }).format(cost);
  };

  const formatNumber = (num: number): string => {
    return new Intl.NumberFormat('en-US').format(num);
  };

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Cost Breakdown by Tags</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <div className="text-muted-foreground">Loading...</div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Cost Breakdown by Tags</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <div className="text-destructive">{error}</div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>Cost Breakdown by Tags</span>
          <div className="flex items-center gap-2">
            <Input
              placeholder="Filter by tag prefix (e.g., org:)"
              value={tagPrefix}
              onChange={(e) => setTagPrefix(e.target.value)}
              className="w-64"
            />
            <Button onClick={handleFilter} variant="outline" size="sm">
              Filter
            </Button>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {data && (
          <>
            {/* Summary Stats */}
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="rounded-lg border p-4">
                <div className="text-sm text-muted-foreground">Total Cost</div>
                <div className="text-2xl font-bold">{formatCost(data.totalCost)}</div>
              </div>
              <div className="rounded-lg border p-4">
                <div className="text-sm text-muted-foreground">Total Tokens</div>
                <div className="text-2xl font-bold">{formatNumber(data.totalTokens)}</div>
              </div>
              <div className="rounded-lg border p-4">
                <div className="text-sm text-muted-foreground">Total Traces</div>
                <div className="text-2xl font-bold">{formatNumber(data.totalTraces)}</div>
              </div>
            </div>

            {/* Breakdown Table */}
            {data.breakdowns.length === 0 ? (
              <div className="flex items-center justify-center py-8 text-muted-foreground">
                No tags found. Add tags to your traces to see cost breakdown.
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tag</TableHead>
                    <TableHead className="text-right">Cost</TableHead>
                    <TableHead className="text-right">Tokens</TableHead>
                    <TableHead className="text-right">Traces</TableHead>
                    <TableHead className="text-right">% of Total</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.breakdowns.map((breakdown) => (
                    <TableRow key={breakdown.tag}>
                      <TableCell>
                        <Badge variant="outline" className="font-mono">
                          {breakdown.tag}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatCost(breakdown.totalCost)}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatNumber(breakdown.totalTokens)}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatNumber(breakdown.traceCount)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <div className="w-16 h-2 bg-muted rounded-full overflow-hidden">
                            <div
                              className="h-full bg-primary rounded-full"
                              style={{ width: `${Math.min(breakdown.percentage, 100)}%` }}
                            />
                          </div>
                          <span className="text-sm text-muted-foreground w-12 text-right">
                            {breakdown.percentage.toFixed(1)}%
                          </span>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
