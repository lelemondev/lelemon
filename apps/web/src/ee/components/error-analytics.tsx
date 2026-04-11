'use client';

import { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { dashboardAPI, ErrorMetrics, ErrorFilter } from '@/lib/api';

interface ErrorAnalyticsProps {
  projectId: string;
}

export function ErrorAnalytics({ projectId }: ErrorAnalyticsProps) {
  const [data, setData] = useState<ErrorMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tagPrefix, setTagPrefix] = useState('');

  const fetchData = useCallback(async (filter?: ErrorFilter) => {
    try {
      setLoading(true);
      setError(null);
      const result = await dashboardAPI.getErrorMetrics(projectId, filter);
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load error metrics');
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

  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleString();
  };

  const getErrorRateColor = (rate: number): string => {
    if (rate >= 20) return 'text-red-500';
    if (rate >= 10) return 'text-orange-500';
    if (rate >= 5) return 'text-yellow-500';
    return 'text-green-500';
  };

  const getErrorRateBg = (rate: number): string => {
    if (rate >= 20) return 'bg-red-500';
    if (rate >= 10) return 'bg-orange-500';
    if (rate >= 5) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Error Analytics</CardTitle>
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
          <CardTitle>Error Analytics</CardTitle>
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
    <div className="space-y-6">
      {/* Error Rate Overview */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Error Rate Overview</span>
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
            <div className="grid grid-cols-3 gap-4">
              <div className="rounded-lg border p-4">
                <div className="text-sm text-muted-foreground">Total Traces</div>
                <div className="text-2xl font-bold">{data.totalTraces.toLocaleString()}</div>
              </div>
              <div className="rounded-lg border p-4">
                <div className="text-sm text-muted-foreground">Error Traces</div>
                <div className="text-2xl font-bold text-red-500">{data.errorTraces.toLocaleString()}</div>
              </div>
              <div className="rounded-lg border p-4">
                <div className="text-sm text-muted-foreground">Error Rate</div>
                <div className={`text-2xl font-bold ${getErrorRateColor(data.errorRate)}`}>
                  {data.errorRate.toFixed(2)}%
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Error Rate by Tag */}
      <Card>
        <CardHeader>
          <CardTitle>Error Rate by Tag</CardTitle>
        </CardHeader>
        <CardContent>
          {data && data.byTag.length === 0 ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              No tags found. Add tags to your traces to see error rate breakdown.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tag</TableHead>
                  <TableHead className="text-right">Total</TableHead>
                  <TableHead className="text-right">Errors</TableHead>
                  <TableHead className="text-right">Error Rate</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.byTag.map((tagRate) => (
                  <TableRow key={tagRate.tag}>
                    <TableCell>
                      <Badge variant="outline" className="font-mono">
                        {tagRate.tag}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      {tagRate.totalTraces.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right text-red-500">
                      {tagRate.errorTraces.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <div className="w-16 h-2 bg-muted rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full ${getErrorRateBg(tagRate.errorRate)}`}
                            style={{ width: `${Math.min(tagRate.errorRate, 100)}%` }}
                          />
                        </div>
                        <span className={`text-sm w-14 text-right ${getErrorRateColor(tagRate.errorRate)}`}>
                          {tagRate.errorRate.toFixed(1)}%
                        </span>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Top Errors */}
      <Card>
        <CardHeader>
          <CardTitle>Top Errors</CardTitle>
        </CardHeader>
        <CardContent>
          {data && data.topErrors.length === 0 ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              No errors found in the selected time range.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Error Message</TableHead>
                  <TableHead className="text-right">Count</TableHead>
                  <TableHead>Last Occurred</TableHead>
                  <TableHead>Affected Tags</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.topErrors.map((errorItem, index) => (
                  <TableRow key={index}>
                    <TableCell className="max-w-md">
                      <div className="font-mono text-sm text-red-500 truncate" title={errorItem.message}>
                        {errorItem.message}
                      </div>
                    </TableCell>
                    <TableCell className="text-right font-bold">
                      {errorItem.count.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDate(errorItem.lastOccurred)}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {errorItem.affectedTags.slice(0, 3).map((tag) => (
                          <Badge key={tag} variant="secondary" className="text-xs">
                            {tag}
                          </Badge>
                        ))}
                        {errorItem.affectedTags.length > 3 && (
                          <Badge variant="secondary" className="text-xs">
                            +{errorItem.affectedTags.length - 3}
                          </Badge>
                        )}
                      </div>
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
