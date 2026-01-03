import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { traces, spans } from '@/db/schema';
import { authenticate, unauthorized, notFound, badRequest } from '@/lib/auth';
import { eq, and, sql } from 'drizzle-orm';
import { calculateCost } from '@/lib/pricing';

const createSpanSchema = z.object({
  parentSpanId: z.string().uuid().optional(),
  type: z.enum(['llm', 'tool', 'retrieval', 'custom']),
  name: z.string().max(100),
  input: z.unknown().optional(),
  output: z.unknown().optional(),
  inputTokens: z.number().int().min(0).optional(),
  outputTokens: z.number().int().min(0).optional(),
  durationMs: z.number().int().min(0).optional(),
  status: z.enum(['pending', 'success', 'error']).optional(),
  errorMessage: z.string().optional(),
  model: z.string().max(50).optional(),
  provider: z.string().max(20).optional(),
  metadata: z.record(z.unknown()).optional(),
  startedAt: z.string().datetime().optional(),
  endedAt: z.string().datetime().optional(),
});

// POST /api/v1/traces/:id/spans - Create span
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  const { id: traceId } = await params;

  try {
    const body = await request.json();
    const result = createSpanSchema.safeParse(body);

    if (!result.success) {
      return badRequest(result.error.message);
    }

    // Verify trace belongs to project
    const trace = await db.query.traces.findFirst({
      where: and(
        eq(traces.id, traceId),
        eq(traces.projectId, auth.projectId)
      ),
      columns: { id: true },
    });

    if (!trace) {
      return notFound('Trace not found');
    }

    const data = result.data;

    // Calculate cost if model and tokens provided
    let costUsd: string | undefined;
    if (data.model && data.inputTokens && data.outputTokens) {
      costUsd = calculateCost(data.model, data.inputTokens, data.outputTokens).toString();
    }

    // Insert span
    const [span] = await db.insert(spans).values({
      traceId,
      parentSpanId: data.parentSpanId,
      type: data.type,
      name: data.name,
      input: data.input,
      output: data.output,
      inputTokens: data.inputTokens,
      outputTokens: data.outputTokens,
      costUsd,
      durationMs: data.durationMs,
      status: data.status || 'success',
      errorMessage: data.errorMessage,
      model: data.model,
      provider: data.provider,
      metadata: data.metadata || {},
      startedAt: data.startedAt ? new Date(data.startedAt) : new Date(),
      endedAt: data.endedAt ? new Date(data.endedAt) : null,
    }).returning({ id: spans.id });

    // Update trace aggregates
    const totalTokens = (data.inputTokens || 0) + (data.outputTokens || 0);
    const costValue = costUsd ? parseFloat(costUsd) : 0;

    await db.update(traces)
      .set({
        totalSpans: sql`${traces.totalSpans} + 1`,
        totalTokens: sql`${traces.totalTokens} + ${totalTokens}`,
        totalCostUsd: sql`${traces.totalCostUsd} + ${costValue}`,
        totalDurationMs: sql`${traces.totalDurationMs} + ${data.durationMs || 0}`,
        updatedAt: new Date(),
      })
      .where(eq(traces.id, traceId));

    return Response.json({ id: span.id }, { status: 201 });
  } catch (error) {
    console.error('Error creating span:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
