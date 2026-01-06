import { NextRequest, NextResponse } from 'next/server';
import { getAgentForProvider } from '@/lib/agents';
import type { Provider } from '@/components/provider-select';

export const runtime = 'nodejs';
export const maxDuration = 60; // 60 seconds timeout

interface ChatRequest {
  provider: Provider;
  message: string;
  sessionId?: string;
}

export async function POST(request: NextRequest) {
  try {
    const body = (await request.json()) as ChatRequest;
    const { provider, message, sessionId } = body;

    // Validate input
    if (!provider || !message) {
      return NextResponse.json(
        { error: 'Missing provider or message' },
        { status: 400 }
      );
    }

    // Validate provider
    const validProviders: Provider[] = ['bedrock', 'anthropic', 'openai', 'gemini'];
    if (!validProviders.includes(provider)) {
      return NextResponse.json(
        { error: `Invalid provider. Must be one of: ${validProviders.join(', ')}` },
        { status: 400 }
      );
    }

    // Check for required API keys
    const apiKeyChecks: Record<Provider, { env: string; name: string }[]> = {
      bedrock: [], // Uses AWS credentials from environment
      anthropic: [{ env: 'ANTHROPIC_API_KEY', name: 'Anthropic' }],
      openai: [{ env: 'OPENAI_API_KEY', name: 'OpenAI' }],
      gemini: [{ env: 'GEMINI_API_KEY', name: 'Gemini' }],
    };

    for (const check of apiKeyChecks[provider]) {
      if (!process.env[check.env]) {
        return NextResponse.json(
          { error: `${check.name} API key not configured. Set ${check.env} in environment.` },
          { status: 500 }
        );
      }
    }

    // Check Lelemon API key
    if (!process.env.LELEMON_API_KEY) {
      return NextResponse.json(
        { error: 'Lelemon API key not configured. Set LELEMON_API_KEY in environment.' },
        { status: 500 }
      );
    }

    // Get agent for provider
    const agent = getAgentForProvider(provider);

    // Run agent with session context
    const result = await agent(message, { sessionId });

    return NextResponse.json(result);
  } catch (error) {
    console.error('[Playground] Chat error:', error);

    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : 'Unknown error occurred',
        details: process.env.NODE_ENV === 'development' && error instanceof Error
          ? error.stack
          : undefined,
      },
      { status: 500 }
    );
  }
}
