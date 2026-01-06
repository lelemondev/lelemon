/**
 * Agent exports
 */

export { runBedrockAgent } from './bedrock';
export { runAnthropicAgent } from './anthropic';
export { runOpenAIAgent } from './openai';
export { runGeminiAgent } from './gemini';
export type { AgentResult, AgentFunction } from './types';

import type { Provider } from '@/components/provider-select';
import { runBedrockAgent } from './bedrock';
import { runAnthropicAgent } from './anthropic';
import { runOpenAIAgent } from './openai';
import { runGeminiAgent } from './gemini';

export function getAgentForProvider(provider: Provider) {
  switch (provider) {
    case 'bedrock':
      return runBedrockAgent;
    case 'anthropic':
      return runAnthropicAgent;
    case 'openai':
      return runOpenAIAgent;
    case 'gemini':
      return runGeminiAgent;
    default:
      throw new Error(`Unknown provider: ${provider}`);
  }
}
