/**
 * LLM Pricing per 1K tokens
 * Prices in USD
 */

interface ModelPricing {
  input: number;  // per 1K tokens
  output: number; // per 1K tokens
}

const PRICING: Record<string, ModelPricing> = {
  // OpenAI
  'gpt-4-turbo': { input: 0.01, output: 0.03 },
  'gpt-4-turbo-preview': { input: 0.01, output: 0.03 },
  'gpt-4o': { input: 0.005, output: 0.015 },
  'gpt-4o-2024-05-13': { input: 0.005, output: 0.015 },
  'gpt-4o-mini': { input: 0.00015, output: 0.0006 },
  'gpt-4o-mini-2024-07-18': { input: 0.00015, output: 0.0006 },
  'gpt-4': { input: 0.03, output: 0.06 },
  'gpt-4-32k': { input: 0.06, output: 0.12 },
  'gpt-3.5-turbo': { input: 0.0005, output: 0.0015 },
  'gpt-3.5-turbo-16k': { input: 0.003, output: 0.004 },
  'o1-preview': { input: 0.015, output: 0.06 },
  'o1-mini': { input: 0.003, output: 0.012 },

  // Anthropic
  'claude-3-5-sonnet-20241022': { input: 0.003, output: 0.015 },
  'claude-3-5-sonnet-latest': { input: 0.003, output: 0.015 },
  'claude-3-opus-20240229': { input: 0.015, output: 0.075 },
  'claude-3-opus-latest': { input: 0.015, output: 0.075 },
  'claude-3-sonnet-20240229': { input: 0.003, output: 0.015 },
  'claude-3-haiku-20240307': { input: 0.00025, output: 0.00125 },

  // AWS Bedrock (Claude)
  'anthropic.claude-3-5-sonnet-20241022-v2:0': { input: 0.003, output: 0.015 },
  'anthropic.claude-3-opus-20240229-v1:0': { input: 0.015, output: 0.075 },
  'anthropic.claude-3-sonnet-20240229-v1:0': { input: 0.003, output: 0.015 },
  'anthropic.claude-3-haiku-20240307-v1:0': { input: 0.00025, output: 0.00125 },

  // Google
  'gemini-1.5-pro': { input: 0.00125, output: 0.005 },
  'gemini-1.5-flash': { input: 0.000075, output: 0.0003 },
  'gemini-2.0-flash': { input: 0.0001, output: 0.0004 },
};

// Default pricing for unknown models
const DEFAULT_PRICING: ModelPricing = { input: 0.001, output: 0.002 };

/**
 * Calculate cost for a model call
 * @returns Cost in USD
 */
export function calculateCost(
  model: string,
  inputTokens: number,
  outputTokens: number
): number {
  const pricing = PRICING[model] || DEFAULT_PRICING;

  const inputCost = (inputTokens / 1000) * pricing.input;
  const outputCost = (outputTokens / 1000) * pricing.output;

  return Number((inputCost + outputCost).toFixed(6));
}

/**
 * Get pricing for a model
 */
export function getModelPricing(model: string): ModelPricing {
  return PRICING[model] || DEFAULT_PRICING;
}
