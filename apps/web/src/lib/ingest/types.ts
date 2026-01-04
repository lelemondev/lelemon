/**
 * Ingest Types
 *
 * Schema definitions for the /api/v1/ingest endpoint.
 * Accepts LLM events directly from the SDK and transforms them into traces + spans.
 */

import { z } from 'zod';

// ─────────────────────────────────────────────────────────────
// Provider Names
// ─────────────────────────────────────────────────────────────

export const ProviderSchema = z.enum([
  'openai',
  'anthropic',
  'gemini',
  'bedrock',
  'openrouter',
  'unknown',
]);

export type Provider = z.infer<typeof ProviderSchema>;

// ─────────────────────────────────────────────────────────────
// Ingest Event Schema
// ─────────────────────────────────────────────────────────────

export const IngestEventSchema = z.object({
  // Provider and model
  provider: ProviderSchema,
  model: z.string().min(1).max(100),

  // Input/Output data
  input: z.unknown(),
  output: z.unknown(),

  // Token counts
  inputTokens: z.number().int().min(0).max(10_000_000),
  outputTokens: z.number().int().min(0).max(10_000_000),

  // Duration
  durationMs: z.number().int().min(0).max(86_400_000), // max 24h

  // Status
  status: z.enum(['success', 'error']),
  errorMessage: z.string().max(10_000).optional(),
  errorStack: z.string().max(50_000).optional(),

  // Streaming flag
  streaming: z.boolean(),

  // Context (for grouping)
  sessionId: z.string().max(255).optional(),
  userId: z.string().max(255).optional(),

  // Custom data
  metadata: z.record(z.unknown()).optional(),
  tags: z.array(z.string().max(50)).max(20).optional(),

  // Optional timestamp (defaults to server time)
  timestamp: z.string().datetime().optional(),
});

export type IngestEvent = z.infer<typeof IngestEventSchema>;

// ─────────────────────────────────────────────────────────────
// Request Schema
// ─────────────────────────────────────────────────────────────

export const IngestRequestSchema = z.object({
  events: z.array(IngestEventSchema).min(1).max(100),
});

export type IngestRequest = z.infer<typeof IngestRequestSchema>;

// ─────────────────────────────────────────────────────────────
// Response Schema
// ─────────────────────────────────────────────────────────────

export interface IngestError {
  index: number;
  message: string;
}

export interface IngestResponse {
  success: boolean;
  processed: number;
  errors?: IngestError[];
}
