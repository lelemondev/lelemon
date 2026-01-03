import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { traces, spans, projects } from '@/db/schema';
import { eq, and, asc } from 'drizzle-orm';
import { createClient } from '@/lib/supabase/server';

// GET /api/v1/dashboard/traces/[id] - Get a trace with its spans (session auth)
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.getUser();

    if (!user?.email) {
      return Response.json({ error: 'Unauthorized' }, { status: 401 });
    }

    // Fetch trace
    const trace = await db.query.traces.findFirst({
      where: eq(traces.id, id),
    });

    if (!trace) {
      return Response.json({ error: 'Trace not found' }, { status: 404 });
    }

    // Verify user owns the project
    const project = await db.query.projects.findFirst({
      where: and(
        eq(projects.id, trace.projectId),
        eq(projects.ownerEmail, user.email)
      ),
    });

    if (!project) {
      return Response.json({ error: 'Trace not found' }, { status: 404 });
    }

    // Fetch spans
    const traceSpans = await db.query.spans.findMany({
      where: eq(spans.traceId, id),
      orderBy: [asc(spans.startedAt)],
    });

    return Response.json({
      ...trace,
      spans: traceSpans,
    });
  } catch (error) {
    console.error('Error fetching trace:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
