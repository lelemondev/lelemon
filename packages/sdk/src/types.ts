/**
 * Lelemon SDK Types
 */

// ============================================
// CONFIG
// ============================================

export interface LelemonConfig {
  /**
   * API key for authentication (starts with 'le_')
   * Can also be set via LELEMON_API_KEY env var
   */
  apiKey?: string;

  /**
   * API endpoint (default: https://api.lelemon.dev)
   */
  endpoint?: string;

  /**
   * Enable debug logging
   */
  debug?: boolean;

  /**
   * Batch size for sending spans (default: 10)
   */
  batchSize?: number;

  /**
   * Flush interval in milliseconds (default: 1000)
   */
  flushInterval?: number;

  /**
   * Disable tracing (useful for testing)
   */
  disabled?: boolean;
}

// ============================================
// TRACE
// ============================================

export interface TraceOptions {
  /**
   * Session ID to group related traces
   */
  sessionId?: string;

  /**
   * User ID for the end user
   */
  userId?: string;

  /**
   * Custom metadata (any JSON-serializable data)
   */
  metadata?: Record<string, unknown>;

  /**
   * Tags for filtering
   */
  tags?: string[];
}

export type TraceStatus = 'active' | 'completed' | 'error';

// ============================================
// SPAN
// ============================================

export type SpanType = 'llm' | 'tool' | 'retrieval' | 'custom';

export type SpanStatus = 'pending' | 'success' | 'error';

export interface SpanOptions {
  /**
   * Type of span
   */
  type: SpanType;

  /**
   * Name of the operation (e.g., 'openai.chat', 'search_documents')
   */
  name: string;

  /**
   * Input data (request body, parameters, etc.)
   */
  input?: unknown;

  /**
   * Custom metadata
   */
  metadata?: Record<string, unknown>;
}

export interface SpanEndOptions {
  /**
   * Output data (response body, result, etc.)
   */
  output?: unknown;

  /**
   * Status of the span
   */
  status?: SpanStatus;

  /**
   * Error message if status is 'error'
   */
  errorMessage?: string;

  /**
   * LLM model used
   */
  model?: string;

  /**
   * LLM provider (openai, anthropic, bedrock)
   */
  provider?: string;

  /**
   * Number of input tokens
   */
  inputTokens?: number;

  /**
   * Number of output tokens
   */
  outputTokens?: number;

  /**
   * Duration in milliseconds (auto-calculated if not provided)
   */
  durationMs?: number;
}

// ============================================
// API TYPES
// ============================================

export interface CreateTraceRequest {
  sessionId?: string;
  userId?: string;
  metadata?: Record<string, unknown>;
  tags?: string[];
}

export interface CreateSpanRequest {
  parentSpanId?: string;
  type: SpanType;
  name: string;
  input?: unknown;
  output?: unknown;
  inputTokens?: number;
  outputTokens?: number;
  durationMs?: number;
  status?: SpanStatus;
  errorMessage?: string;
  model?: string;
  provider?: string;
  metadata?: Record<string, unknown>;
  startedAt?: string;
  endedAt?: string;
}

export interface UpdateTraceRequest {
  status?: TraceStatus;
  metadata?: Record<string, unknown>;
  tags?: string[];
}
