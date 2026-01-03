import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { traces } from '@/db/schema';
import { authenticate, unauthorized, badRequest } from '@/lib/auth';
import { eq, and, desc, sql } from 'drizzle-orm';

// GET /api/v1/traces - List traces
export async function GET(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  const { searchParams } = new URL(request.url);
  const limit = Math.min(parseInt(searchParams.get('limit') || '50'), 100);
  const offset = parseInt(searchParams.get('offset') || '0');
  const sessionId = searchParams.get('sessionId');
  const userId = searchParams.get('userId');
  const status = searchParams.get('status');

  try {
    const conditions = [eq(traces.projectId, auth.projectId)];

    if (sessionId) {
      conditions.push(eq(traces.sessionId, sessionId));
    }
    if (userId) {
      conditions.push(eq(traces.userId, userId));
    }
    if (status && ['active', 'completed', 'error'].includes(status)) {
      conditions.push(eq(traces.status, status as 'active' | 'completed' | 'error'));
    }

    const [data, countResult] = await Promise.all([
      db.query.traces.findMany({
        where: and(...conditions),
        orderBy: [desc(traces.createdAt)],
        limit,
        offset,
      }),
      db.select({ count: sql<number>`count(*)` })
        .from(traces)
        .where(and(...conditions)),
    ]);

    return Response.json({
      data,
      total: Number(countResult[0]?.count || 0),
      limit,
      offset,
    });
  } catch (error) {
    console.error('Error fetching traces:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}

// POST /api/v1/traces - Create trace
const createTraceSchema = z.object({
  sessionId: z.string().max(100).optional(),
  userId: z.string().max(100).optional(),
  metadata: z.record(z.unknown()).optional(),
  tags: z.array(z.string().max(50)).optional(),
});

export async function POST(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  try {
    const body = await request.json();
    const result = createTraceSchema.safeParse(body);

    if (!result.success) {
      return badRequest(result.error.message);
    }

    const { sessionId, userId, metadata, tags } = result.data;

    const [trace] = await db.insert(traces).values({
      projectId: auth.projectId,
      sessionId,
      userId,
      metadata: metadata || {},
      tags,
    }).returning({ id: traces.id });

    return Response.json({ id: trace.id }, { status: 201 });
  } catch (error) {
    console.error('Error creating trace:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
