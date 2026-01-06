/**
 * Common types for all agents
 */

export interface AgentOptions {
  sessionId?: string;
  userId?: string;
}

export interface AgentResult {
  response: string;
  traceId?: string;
  sessionId?: string;
  provider: string;
  model: string;
  durationMs: number;
  toolsUsed?: string[];
}

export type AgentFunction = (message: string, options?: AgentOptions) => Promise<AgentResult>;
