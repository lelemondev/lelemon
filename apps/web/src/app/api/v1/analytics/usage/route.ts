import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { traces } from '@/db/schema';
import { authenticate, unauthorized } from '@/lib/auth';
import { eq, and, gte, lte, sql } from 'drizzle-orm';

// GET /api/v1/analytics/usage
export async function GET(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  const { searchParams } = new URL(request.url);
  const from = searchParams.get('from') || new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
  const to = searchParams.get('to') || new Date().toISOString();
  const groupBy = searchParams.get('groupBy') || 'day';

  try {
    const dateFormat = groupBy === 'hour' ? 'YYYY-MM-DD HH24:00' : 'YYYY-MM-DD';

    const result = await db.select({
      date: sql<string>`to_char(${traces.createdAt}, ${dateFormat})`,
      traces: sql<number>`count(*)`,
      spans: sql<number>`sum(${traces.totalSpans})`,
      tokens: sql<number>`sum(${traces.totalTokens})`,
      costUsd: sql<number>`sum(${traces.totalCostUsd})`,
    })
    .from(traces)
    .where(and(
      eq(traces.projectId, auth.projectId),
      gte(traces.createdAt, new Date(from)),
      lte(traces.createdAt, new Date(to))
    ))
    .groupBy(sql`to_char(${traces.createdAt}, ${dateFormat})`)
    .orderBy(sql`to_char(${traces.createdAt}, ${dateFormat})`);

    return Response.json(result.map(row => ({
      date: row.date,
      traces: Number(row.traces || 0),
      spans: Number(row.spans || 0),
      tokens: Number(row.tokens || 0),
      costUsd: Number(row.costUsd || 0),
    })));
  } catch (error) {
    console.error('Error fetching usage analytics:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
