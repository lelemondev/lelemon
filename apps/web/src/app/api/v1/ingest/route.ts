/**
 * POST /api/v1/ingest
 *
 * Accepts LLM events directly from the SDK.
 * Creates traces and spans automatically based on sessionId grouping.
 */

import { NextRequest } from 'next/server';
import { authenticate, unauthorized, badRequest, checkProjectRateLimit } from '@/lib/auth';
import { IngestRequestSchema } from '@/lib/ingest/types';
import { processIngestEvents } from '@/lib/ingest/processor';

export async function POST(request: NextRequest) {
  // Authenticate
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  // Rate limit
  const rateLimited = checkProjectRateLimit(auth.projectId);
  if (rateLimited) return rateLimited;

  try {
    const body = await request.json();
    const result = IngestRequestSchema.safeParse(body);

    if (!result.success) {
      return badRequest(result.error.message);
    }

    const response = await processIngestEvents(auth.projectId, result.data.events);

    return Response.json(response, {
      status: response.success ? 200 : 207, // 207 Multi-Status for partial success
    });
  } catch (error) {
    console.error('[Ingest] Error:', error);
    return Response.json(
      { success: false, processed: 0, errors: [{ index: -1, message: 'Internal server error' }] },
      { status: 500 }
    );
  }
}
