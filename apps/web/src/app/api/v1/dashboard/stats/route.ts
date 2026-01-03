import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { traces, projects } from '@/db/schema';
import { eq, and, sql } from 'drizzle-orm';
import { createClient } from '@/lib/supabase/server';

// GET /api/v1/dashboard/stats - Get project statistics (session auth)
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

    // Calculate stats
    const statsResult = await db
      .select({
        totalTraces: sql<number>`count(*)::int`,
        totalTokens: sql<number>`coalesce(sum(${traces.totalTokens}), 0)::int`,
        totalCostUsd: sql<number>`coalesce(sum(${traces.totalCostUsd}::numeric), 0)::numeric`,
        errorCount: sql<number>`count(*) filter (where ${traces.status} = 'error')::int`,
      })
      .from(traces)
      .where(eq(traces.projectId, projectId));

    const stats = statsResult[0];
    const errorRate = stats.totalTraces > 0
      ? (stats.errorCount / stats.totalTraces) * 100
      : 0;

    return Response.json({
      totalTraces: stats.totalTraces,
      totalTokens: stats.totalTokens,
      totalCostUsd: parseFloat(String(stats.totalCostUsd)),
      errorRate,
    });
  } catch (error) {
    console.error('Error fetching stats:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
