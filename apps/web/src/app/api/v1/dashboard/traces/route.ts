import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { traces, projects } from '@/db/schema';
import { eq, desc, and } from 'drizzle-orm';
import { createClient } from '@/lib/supabase/server';

// GET /api/v1/dashboard/traces - List traces for a project (session auth)
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

    // Fetch traces
    const projectTraces = await db.query.traces.findMany({
      where: eq(traces.projectId, projectId),
      orderBy: [desc(traces.createdAt)],
      limit: 100,
    });

    return Response.json(projectTraces);
  } catch (error) {
    console.error('Error fetching traces:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
