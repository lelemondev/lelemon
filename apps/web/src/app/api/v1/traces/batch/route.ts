import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { traces, spans } from '@/db/schema';
import { authenticate, unauthorized, badRequest, checkProjectRateLimit } from '@/lib/auth';
import { calculateCost } from '@/lib/pricing';
import { eq, and } from 'drizzle-orm';

// ─────────────────────────────────────────────────────────────
// Schema validation
// ─────────────────────────────────────────────────────────────

const llmCallSchema = z.object({
  model: z.string().optional(),
  provider: z.string().optional(),
  inputTokens: z.number().optional(),
  outputTokens: z.number().optional(),
  input: z.unknown().optional(),
  output: z.unknown().optional(),
});

const toolCallSchema = z.object({
  name: z.string(),
  input: z.unknown(),
  output: z.unknown().optional(),
});

const createTraceSchema = z.object({
  tempId: z.string(),
  data: z.object({
    name: z.string().max(100).optional(),
    sessionId: z.string().max(100).optional(),
    userId: z.string().max(100).optional(),
    input: z.unknown().optional(),
    metadata: z.record(z.unknown()).optional(),
    tags: z.array(z.string().max(50)).optional(),
  }),
});

const completeTraceSchema = z.object({
  traceId: z.string().uuid(),
  data: z.object({
    status: z.enum(['completed', 'error']),
    output: z.unknown().optional(),
    errorMessage: z.string().optional(),
    errorStack: z.string().optional(),
    systemPrompt: z.string().optional(),
    llmCalls: z.array(llmCallSchema).optional(),
    toolCalls: z.array(toolCallSchema).optional(),
    models: z.array(z.string()).optional(),
    totalInputTokens: z.number().optional(),
    totalOutputTokens: z.number().optional(),
    durationMs: z.number().optional(),
    metadata: z.record(z.unknown()).optional(),
  }),
});

const batchRequestSchema = z.object({
  creates: z.array(createTraceSchema).optional().default([]),
  completes: z.array(completeTraceSchema).optional().default([]),
});

// ─────────────────────────────────────────────────────────────
// POST /api/v1/traces/batch - Batch create and complete traces
// ─────────────────────────────────────────────────────────────

export async function POST(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  // Rate limit by project (100 req/min)
  const rateLimited = checkProjectRateLimit(auth.projectId);
  if (rateLimited) return rateLimited;

  try {
    const body = await request.json();
    const result = batchRequestSchema.safeParse(body);

    if (!result.success) {
      return badRequest(result.error.message);
    }

    const { creates, completes } = result.data;
    const createdMapping: Record<string, string> = {};
    const errors: string[] = [];

    // ─────────────────────────────────────────────────────────
    // Process creates
    // ─────────────────────────────────────────────────────────

    if (creates.length > 0) {
      const createValues = creates.map((c) => ({
        projectId: auth.projectId,
        sessionId: c.data.sessionId,
        userId: c.data.userId,
        metadata: {
          ...(c.data.metadata || {}),
          name: c.data.name,
          input: c.data.input,
        },
        tags: c.data.tags,
      }));

      const createdTraces = await db.insert(traces)
        .values(createValues)
        .returning({ id: traces.id });

      // Map tempId to real ID
      creates.forEach((c, i) => {
        if (createdTraces[i]) {
          createdMapping[c.tempId] = createdTraces[i].id;
        }
      });
    }

    // ─────────────────────────────────────────────────────────
    // Process completes
    // ─────────────────────────────────────────────────────────

    for (const complete of completes) {
      try {
        await processComplete(auth.projectId, complete);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        errors.push(`Trace ${complete.traceId}: ${message}`);
      }
    }

    return Response.json({
      created: createdMapping,
      errors: errors.length > 0 ? errors : undefined,
    });
  } catch (error) {
    console.error('Batch error:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}

// ─────────────────────────────────────────────────────────────
// Process a trace completion
// ─────────────────────────────────────────────────────────────

async function processComplete(
  projectId: string,
  complete: z.infer<typeof completeTraceSchema>
): Promise<void> {
  const { traceId, data } = complete;

  // Verify trace belongs to project
  const existing = await db.query.traces.findFirst({
    where: and(
      eq(traces.id, traceId),
      eq(traces.projectId, projectId)
    ),
    columns: { id: true },
  });

  if (!existing) {
    throw new Error('Not found');
  }

  // Calculate total cost and tokens
  let totalCost = 0;
  const totalTokens = (data.totalInputTokens || 0) + (data.totalOutputTokens || 0);
  const spansToCreate: Array<typeof spans.$inferInsert> = [];

  // Process LLM calls into spans
  if (data.llmCalls && data.llmCalls.length > 0) {
    for (const call of data.llmCalls) {
      const inputTokens = call.inputTokens || 0;
      const outputTokens = call.outputTokens || 0;
      const cost = call.model
        ? calculateCost(call.model, inputTokens, outputTokens)
        : 0;

      totalCost += cost;

      spansToCreate.push({
        traceId,
        type: 'llm',
        name: call.model || 'llm-call',
        input: call.input,
        output: call.output,
        inputTokens,
        outputTokens,
        costUsd: String(cost),
        model: call.model,
        provider: call.provider,
        status: 'success',
        startedAt: new Date(),
        endedAt: new Date(),
      });
    }
  }

  // Process tool calls into spans
  if (data.toolCalls && data.toolCalls.length > 0) {
    for (const tool of data.toolCalls) {
      spansToCreate.push({
        traceId,
        type: 'tool',
        name: tool.name,
        input: tool.input,
        output: tool.output,
        status: 'success',
        startedAt: new Date(),
        endedAt: new Date(),
      });
    }
  }

  // Insert spans if any
  if (spansToCreate.length > 0) {
    await db.insert(spans).values(spansToCreate);
  }

  // Update trace
  await db.update(traces)
    .set({
      status: data.status,
      totalTokens,
      totalCostUsd: String(totalCost),
      totalDurationMs: data.durationMs || 0,
      totalSpans: spansToCreate.length,
      metadata: {
        output: data.output,
        systemPrompt: data.systemPrompt,
        errorMessage: data.errorMessage,
        errorStack: data.errorStack,
        models: data.models,
        ...(data.metadata || {}),
      },
      updatedAt: new Date(),
    })
    .where(eq(traces.id, traceId));
}
