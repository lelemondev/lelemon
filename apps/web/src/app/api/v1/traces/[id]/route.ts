import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { traces, spans } from '@/db/schema';
import { authenticate, unauthorized, notFound, badRequest } from '@/lib/auth';
import { eq, and, asc } from 'drizzle-orm';

// GET /api/v1/traces/:id - Get trace with spans
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  const { id } = await params;

  try {
    const trace = await db.query.traces.findFirst({
      where: and(
        eq(traces.id, id),
        eq(traces.projectId, auth.projectId)
      ),
    });

    if (!trace) {
      return notFound('Trace not found');
    }

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

// PATCH /api/v1/traces/:id - Update trace
const updateTraceSchema = z.object({
  status: z.enum(['active', 'completed', 'error']).optional(),
  metadata: z.record(z.unknown()).optional(),
  tags: z.array(z.string().max(50)).optional(),
});

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  const { id } = await params;

  try {
    const body = await request.json();
    const result = updateTraceSchema.safeParse(body);

    if (!result.success) {
      return badRequest(result.error.message);
    }

    // Verify trace belongs to project
    const existing = await db.query.traces.findFirst({
      where: and(
        eq(traces.id, id),
        eq(traces.projectId, auth.projectId)
      ),
      columns: { id: true },
    });

    if (!existing) {
      return notFound('Trace not found');
    }

    const { status, metadata, tags } = result.data;

    await db.update(traces)
      .set({
        ...(status && { status }),
        ...(metadata && { metadata }),
        ...(tags && { tags }),
        updatedAt: new Date(),
      })
      .where(eq(traces.id, id));

    return new Response(null, { status: 204 });
  } catch (error) {
    console.error('Error updating trace:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
