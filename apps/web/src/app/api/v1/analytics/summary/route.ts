import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { traces } from '@/db/schema';
import { authenticate, unauthorized } from '@/lib/auth';
import { eq, and, gte, lte, sql } from 'drizzle-orm';

// GET /api/v1/analytics/summary
export async function GET(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  const { searchParams } = new URL(request.url);
  const from = searchParams.get('from');
  const to = searchParams.get('to');

  try {
    const conditions = [eq(traces.projectId, auth.projectId)];

    if (from) {
      conditions.push(gte(traces.createdAt, new Date(from)));
    }
    if (to) {
      conditions.push(lte(traces.createdAt, new Date(to)));
    }

    const result = await db.select({
      totalTraces: sql<number>`count(*)`,
      totalSpans: sql<number>`sum(${traces.totalSpans})`,
      totalTokens: sql<number>`sum(${traces.totalTokens})`,
      totalCostUsd: sql<number>`sum(${traces.totalCostUsd})`,
      avgDurationMs: sql<number>`avg(${traces.totalDurationMs})`,
      errorCount: sql<number>`sum(case when ${traces.status} = 'error' then 1 else 0 end)`,
    })
    .from(traces)
    .where(and(...conditions));

    const data = result[0];
    const totalTraces = Number(data?.totalTraces || 0);
    const errorCount = Number(data?.errorCount || 0);

    return Response.json({
      totalTraces,
      totalSpans: Number(data?.totalSpans || 0),
      totalTokens: Number(data?.totalTokens || 0),
      totalCostUsd: Number(data?.totalCostUsd || 0),
      avgDurationMs: Number(data?.avgDurationMs || 0),
      errorRate: totalTraces > 0 ? (errorCount / totalTraces) * 100 : 0,
    });
  } catch (error) {
    console.error('Error fetching analytics summary:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
