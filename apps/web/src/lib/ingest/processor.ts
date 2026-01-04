/**
 * Ingest Processor
 *
 * Processes incoming LLM events from the SDK.
 * Groups events by sessionId and creates traces + spans.
 */

import { db } from '@/db/client';
import { traces, spans } from '@/db/schema';
import { calculateCost } from '@/lib/pricing';
import { eq, and, desc } from 'drizzle-orm';
import type { IngestEvent, IngestResponse, IngestError } from './types';

// ─────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────

interface IndexedEvent extends IngestEvent {
  _index: number;
}

// ─────────────────────────────────────────────────────────────
// Main Processor
// ─────────────────────────────────────────────────────────────

/**
 * Process a batch of ingest events
 */
export async function processIngestEvents(
  projectId: string,
  events: IngestEvent[]
): Promise<IngestResponse> {
  const errors: IngestError[] = [];
  let processed = 0;

  // Add index to each event for error tracking
  const indexedEvents: IndexedEvent[] = events.map((event, index) => ({
    ...event,
    _index: index,
  }));

  // Group events by sessionId
  const eventsBySession = groupBySessionId(indexedEvents);

  // Process each session group
  for (const [sessionId, sessionEvents] of eventsBySession) {
    try {
      await processSessionEvents(projectId, sessionId, sessionEvents);
      processed += sessionEvents.length;
    } catch (error) {
      // Record errors but continue with other events
      const message = error instanceof Error ? error.message : 'Unknown error';
      for (const event of sessionEvents) {
        errors.push({ index: event._index, message });
      }
    }
  }

  return {
    success: errors.length === 0,
    processed,
    errors: errors.length > 0 ? errors : undefined,
  };
}

// ─────────────────────────────────────────────────────────────
// Session Processing
// ─────────────────────────────────────────────────────────────

/**
 * Process events for a single session
 */
async function processSessionEvents(
  projectId: string,
  sessionId: string | null,
  events: IndexedEvent[]
): Promise<void> {
  // Get or create trace
  const traceId = await getOrCreateTrace(projectId, sessionId, events[0]);

  // Accumulators for trace totals
  let totalInputTokens = 0;
  let totalOutputTokens = 0;
  let totalCostUsd = 0;
  let totalDurationMs = 0;

  // Create span for each event
  const spansToInsert = events.map((event) => {
    const cost = calculateCost(event.model, event.inputTokens, event.outputTokens);

    totalInputTokens += event.inputTokens;
    totalOutputTokens += event.outputTokens;
    totalCostUsd += cost;
    totalDurationMs += event.durationMs;

    return {
      traceId,
      type: 'llm' as const,
      name: event.model,
      provider: event.provider,
      model: event.model,
      input: event.input,
      output: event.output,
      inputTokens: event.inputTokens,
      outputTokens: event.outputTokens,
      costUsd: String(cost),
      durationMs: event.durationMs,
      status: event.status === 'error' ? ('error' as const) : ('success' as const),
      errorMessage: event.errorMessage,
      metadata: {
        streaming: event.streaming,
        ...(event.metadata || {}),
      },
      startedAt: event.timestamp ? new Date(event.timestamp) : new Date(),
      endedAt: new Date(),
    };
  });

  // Insert spans
  if (spansToInsert.length > 0) {
    await db.insert(spans).values(spansToInsert);
  }

  // Determine final status
  const hasErrors = events.some((e) => e.status === 'error');
  const finalStatus = hasErrors ? 'error' : 'completed';

  // Update trace totals
  const lastEvent = events[events.length - 1];
  await db
    .update(traces)
    .set({
      status: finalStatus,
      totalTokens: totalInputTokens + totalOutputTokens,
      totalCostUsd: String(totalCostUsd),
      totalDurationMs: totalDurationMs,
      totalSpans: spansToInsert.length,
      metadata: {
        lastOutput: lastEvent.output,
        ...(lastEvent.metadata || {}),
      },
      updatedAt: new Date(),
    })
    .where(eq(traces.id, traceId));
}

// ─────────────────────────────────────────────────────────────
// Trace Management
// ─────────────────────────────────────────────────────────────

/**
 * Get existing trace by sessionId or create a new one
 */
async function getOrCreateTrace(
  projectId: string,
  sessionId: string | null,
  firstEvent: IndexedEvent
): Promise<string> {
  // If we have a sessionId, try to find existing trace
  if (sessionId) {
    const existing = await db.query.traces.findFirst({
      where: and(
        eq(traces.projectId, projectId),
        eq(traces.sessionId, sessionId)
      ),
      orderBy: [desc(traces.createdAt)],
      columns: { id: true },
    });

    if (existing) {
      return existing.id;
    }
  }

  // Create new trace
  const [newTrace] = await db
    .insert(traces)
    .values({
      projectId,
      sessionId: sessionId || undefined,
      userId: firstEvent.userId,
      metadata: {
        input: firstEvent.input,
        ...(firstEvent.metadata || {}),
      },
      tags: firstEvent.tags,
      status: 'active',
    })
    .returning({ id: traces.id });

  return newTrace.id;
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

/**
 * Group events by sessionId
 */
function groupBySessionId(
  events: IndexedEvent[]
): Map<string | null, IndexedEvent[]> {
  const groups = new Map<string | null, IndexedEvent[]>();

  for (const event of events) {
    const key = event.sessionId ?? null;
    const existing = groups.get(key) ?? [];
    existing.push(event);
    groups.set(key, existing);
  }

  return groups;
}
