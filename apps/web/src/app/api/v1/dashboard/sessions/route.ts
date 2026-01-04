import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { traces, projects } from '@/db/schema';
import { eq, and, sql, desc } from 'drizzle-orm';
import { createClient } from '@/lib/supabase/server';

// GET /api/v1/dashboard/sessions - List sessions with aggregated metrics
export async function GET(request: NextRequest) {
  try {
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.getUser();

    if (!user?.email) {
      return Response.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const { searchParams } = new URL(request.url);
    const projectId = searchParams.get('projectId');

    if (!projectId) {
      return Response.json({ error: 'projectId is required' }, { status: 400 });
    }

    // Verify user owns this project
    const project = await db.query.projects.findFirst({
      where: and(
        eq(projects.id, projectId),
        eq(projects.ownerEmail, user.email)
      ),
    });

    if (!project) {
      return Response.json({ error: 'Project not found' }, { status: 404 });
    }

    // Aggregate traces by sessionId
    const sessions = await db
      .select({
        sessionId: traces.sessionId,
        userId: sql<string>`MAX(${traces.userId})`.as('userId'),
        traceCount: sql<number>`COUNT(*)::int`.as('trace_count'),
        totalTokens: sql<number>`SUM(${traces.totalTokens})::int`.as('total_tokens'),
        totalCostUsd: sql<string>`SUM(${traces.totalCostUsd})`.as('total_cost_usd'),
        totalDurationMs: sql<number>`SUM(${traces.totalDurationMs})::int`.as('total_duration_ms'),
        totalSpans: sql<number>`SUM(${traces.totalSpans})::int`.as('total_spans'),
        hasError: sql<boolean>`BOOL_OR(${traces.status} = 'error')`.as('has_error'),
        hasActive: sql<boolean>`BOOL_OR(${traces.status} = 'active')`.as('has_active'),
        firstTraceAt: sql<string>`MIN(${traces.createdAt})`.as('first_trace_at'),
        lastTraceAt: sql<string>`MAX(${traces.createdAt})`.as('last_trace_at'),
      })
      .from(traces)
      .where(and(
        eq(traces.projectId, projectId),
        sql`${traces.sessionId} IS NOT NULL`
      ))
      .groupBy(traces.sessionId)
      .orderBy(desc(sql`MAX(${traces.createdAt})`))
      .limit(100);

    return Response.json(sessions);
  } catch (error) {
    console.error('Error fetching sessions:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
