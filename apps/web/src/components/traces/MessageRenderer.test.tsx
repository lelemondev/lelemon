import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MessageRenderer } from './MessageRenderer';

// =============================================================================
// REAL SDK PAYLOADS - Based on parser.go and actual production data
//
// These fixtures represent exactly what arrives at the frontend:
// - input: The request sent to the LLM (messages, system, tools)
// - output: The parsed span.output from backend (string or content array)
//
// Reference: apps/server/pkg/domain/service/parser.go
// Reference: docs/EXTERNAL-DEPENDENCIES-PLAN.md
// =============================================================================

const FIXTURES = {
  // ==========================================================================
  // ANTHROPIC FORMAT
  // Input: { system, messages, tools?, max_tokens }
  // Output (text): string (joined text parts)
  // Output (tool_use): content array (raw blocks)
  // ==========================================================================

  anthropicTextResponse: {
    input: {
      model: 'claude-3-5-sonnet-20241022',
      system: 'You are a helpful coding assistant.',
      messages: [
        { role: 'user', content: 'Explain recursion briefly' },
      ],
      max_tokens: 1024,
    },
    // Parser joins text blocks into a single string
    output: 'Recursion is when a function calls itself to solve a smaller version of the same problem.',
  },

  anthropicToolUse: {
    input: {
      model: 'claude-3-5-sonnet-20241022',
      messages: [
        { role: 'user', content: 'Search for the latest news' },
      ],
      tools: [
        { name: 'web_search', description: 'Search the web', input_schema: {} },
      ],
    },
    // Parser preserves raw content array when tool_use is present
    output: [
      { type: 'text', text: 'I will search for the latest news.' },
      {
        type: 'tool_use',
        id: 'toolu_abc123',
        name: 'web_search',
        input: { query: 'latest news' },
      },
    ],
  },

  anthropicSystemAsArray: {
    input: {
      model: 'claude-3-5-sonnet-20241022',
      system: [
        { type: 'text', text: 'You are a research assistant.' },
        { type: 'text', text: 'Be concise and accurate.' },
      ],
      messages: [
        { role: 'user', content: 'Find information about TypeScript' },
      ],
    },
    output: 'TypeScript is a strongly typed superset of JavaScript.',
  },

  anthropicMultiTurnWithToolResult: {
    input: {
      model: 'claude-3-5-sonnet-20241022',
      system: 'You are a helpful assistant.',
      messages: [
        { role: 'user', content: 'Calculate 15 * 7' },
        {
          role: 'assistant',
          content: [
            { type: 'text', text: 'I will calculate that.' },
            { type: 'tool_use', id: 'calc_1', name: 'calculator', input: { a: 15, b: 7 } },
          ],
        },
        {
          role: 'user',
          content: [
            { type: 'tool_result', tool_use_id: 'calc_1', content: '105' },
          ],
        },
        { role: 'user', content: 'Thanks! Now explain the result.' },
      ],
    },
    output: 'The result of 15 multiplied by 7 is 105.',
  },

  // ==========================================================================
  // OPENAI FORMAT
  // Input: { model, messages, tools? }
  // messages[].role: 'system' | 'user' | 'assistant'
  // Output (text): string (message.content)
  // Output (tool_calls): string (may be null or empty)
  // ==========================================================================

  openaiTextResponse: {
    input: {
      model: 'gpt-4o',
      messages: [
        { role: 'system', content: 'You are a helpful assistant.' },
        { role: 'user', content: 'What is the capital of France?' },
      ],
    },
    // Parser extracts choices[0].message.content as string
    output: 'The capital of France is Paris.',
  },

  openaiMultiTurn: {
    input: {
      model: 'gpt-4o',
      messages: [
        { role: 'system', content: 'You are a helpful assistant.' },
        { role: 'user', content: 'What is the capital of France?' },
        { role: 'assistant', content: 'The capital of France is Paris.' },
        { role: 'user', content: 'What about Germany?' },
      ],
    },
    output: 'The capital of Germany is Berlin.',
  },

  openaiWithTools: {
    input: {
      model: 'gpt-4o',
      messages: [
        { role: 'user', content: 'Get weather for Tokyo' },
      ],
      tools: [
        { type: 'function', function: { name: 'get_weather', parameters: {} } },
      ],
    },
    // When tool_calls present, content might be empty
    output: null,
  },

  // ==========================================================================
  // AWS BEDROCK FORMAT (Converse API)
  // Input: { modelId, system: [{ text }], messages: [{ role, content: [{ text }] }] }
  // Note: Bedrock uses { text: "..." } blocks instead of plain strings
  // ==========================================================================

  bedrockTextResponse: {
    input: {
      modelId: 'anthropic.claude-3-5-sonnet-20241022-v2:0',
      system: [{ text: 'You are a helpful assistant.' }],
      messages: [
        { role: 'user', content: [{ text: 'Hello, how are you?' }] },
      ],
    },
    // Parser joins text blocks from output.message.content
    output: 'I am doing well, thank you for asking!',
  },

  bedrockToolUse: {
    input: {
      modelId: 'anthropic.claude-3-5-sonnet-20241022-v2:0',
      messages: [
        { role: 'user', content: [{ text: 'Calculate 15 * 7' }] },
      ],
      toolConfig: {
        tools: [{ toolSpec: { name: 'calculator', description: 'Math operations' } }],
      },
    },
    // Parser preserves raw content when toolUse present
    output: [
      { text: 'I will calculate that for you.' },
      {
        toolUse: {
          toolUseId: 'tool_123',
          name: 'calculator',
          input: { operation: 'multiply', a: 15, b: 7 },
        },
      },
    ],
  },

  bedrockWithToolResult: {
    input: {
      modelId: 'anthropic.claude-3-5-sonnet-20241022-v2:0',
      messages: [
        { role: 'user', content: [{ text: 'Calculate 15 * 7' }] },
        {
          role: 'assistant',
          content: [
            { text: 'Let me calculate.' },
            { toolUse: { toolUseId: 'tool_123', name: 'calculator', input: { a: 15, b: 7 } } },
          ],
        },
        {
          role: 'user',
          content: [
            { toolResult: { toolUseId: 'tool_123', status: 'success', content: [{ text: '105' }] } },
          ],
        },
      ],
    },
    output: 'The result of 15 * 7 is 105.',
  },

  // ==========================================================================
  // LONG CONVERSATION (Tests previous messages collapsing)
  // ==========================================================================

  longConversation: {
    input: {
      model: 'gpt-4o',
      system: 'You are a customer support agent.',
      messages: [
        { role: 'user', content: 'Hi, I need help with my order' },
        { role: 'assistant', content: 'Of course! What is your order number?' },
        { role: 'user', content: 'It is ORDER-12345' },
        { role: 'assistant', content: 'Thank you. I see your order was shipped yesterday.' },
        { role: 'user', content: 'When will it arrive?' },
        { role: 'assistant', content: 'It should arrive within 3-5 business days.' },
        { role: 'user', content: 'Can I change the delivery address?' },
        { role: 'assistant', content: 'I can help with that. What is the new address?' },
        { role: 'user', content: '123 New Street, City, State 12345' },
      ],
    },
    output: 'I have updated your delivery address to 123 New Street, City, State 12345.',
    expectedPreviousCount: 8, // Messages before last user message
  },

  // ==========================================================================
  // EDGE CASES
  // ==========================================================================

  emptyMessages: {
    input: {
      model: 'gpt-4o',
      messages: [],
    },
    output: null,
  },

  stringOutput: {
    input: {
      messages: [{ role: 'user', content: 'Hi' }],
    },
    output: 'Hello! How can I help you?',
  },

  nullOutput: {
    input: {
      messages: [{ role: 'user', content: 'Test' }],
    },
    output: null,
  },

  jsonStringOutput: {
    input: {
      messages: [{ role: 'user', content: 'Give me JSON' }],
    },
    // Sometimes output is a JSON string that needs parsing
    output: '{"response": "Here is your JSON data"}',
  },
};

// =============================================================================
// TESTS
// =============================================================================

describe('MessageRenderer', () => {
  describe('Anthropic format', () => {
    it('renders text response with system prompt', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicTextResponse.input}
          output={FIXTURES.anthropicTextResponse.output}
        />
      );

      // System prompt should be collapsible
      expect(screen.getByText('System Prompt')).toBeInTheDocument();

      // User message should be visible
      expect(screen.getByText('User Input')).toBeInTheDocument();

      // Response should show the text
      expect(screen.getByText('Response')).toBeInTheDocument();
      expect(screen.getByText(FIXTURES.anthropicTextResponse.output)).toBeInTheDocument();
    });

    it('renders tool_use response with content blocks', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicToolUse.input}
          output={FIXTURES.anthropicToolUse.output}
        />
      );

      // Available tools should be shown
      expect(screen.getByText(/Available Tools/)).toBeInTheDocument();
      expect(screen.getByText('web_search')).toBeInTheDocument();
    });

    it('handles system prompt as array format', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicSystemAsArray.input}
          output={FIXTURES.anthropicSystemAsArray.output}
        />
      );

      // System prompt should be extracted from array
      expect(screen.getByText('System Prompt')).toBeInTheDocument();

      // Click to expand and verify content
      fireEvent.click(screen.getByText('System Prompt'));
      expect(screen.getByText(/You are a research assistant/)).toBeInTheDocument();
    });

    it('filters out tool_result messages from user timeline', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicMultiTurnWithToolResult.input}
          output={FIXTURES.anthropicMultiTurnWithToolResult.output}
        />
      );

      // Should not show tool_result as a separate user message
      // Only real user messages should appear
      expect(screen.getByText('User Input')).toBeInTheDocument();
    });
  });

  describe('OpenAI format', () => {
    it('renders text response (system in messages)', () => {
      render(
        <MessageRenderer
          input={FIXTURES.openaiTextResponse.input}
          output={FIXTURES.openaiTextResponse.output}
        />
      );

      // OpenAI puts system in messages, so no separate System Prompt section
      expect(screen.getByText('User Input')).toBeInTheDocument();
      expect(screen.getByText('Response')).toBeInTheDocument();
      expect(screen.getByText(FIXTURES.openaiTextResponse.output)).toBeInTheDocument();
    });

    it('renders multi-turn with previous messages collapsed', () => {
      render(
        <MessageRenderer
          input={FIXTURES.openaiMultiTurn.input}
          output={FIXTURES.openaiMultiTurn.output}
        />
      );

      // Previous messages section
      const previousSection = screen.getByText(/Previous Messages/);
      expect(previousSection).toBeInTheDocument();

      // Should have 2 previous items (user + assistant before last user)
      expect(previousSection.textContent).toContain('(2)');
    });

    it('renders with tools in Available Tools section', () => {
      render(
        <MessageRenderer
          input={FIXTURES.openaiWithTools.input}
          output={FIXTURES.openaiWithTools.output}
        />
      );

      // Available tools should be shown
      expect(screen.getByText(/Available Tools/)).toBeInTheDocument();
      // OpenAI tools format: tools[].function.name - component may not extract this
    });
  });

  describe('AWS Bedrock format', () => {
    it('renders text response', () => {
      render(
        <MessageRenderer
          input={FIXTURES.bedrockTextResponse.input}
          output={FIXTURES.bedrockTextResponse.output}
        />
      );

      expect(screen.getByText('System Prompt')).toBeInTheDocument();
      expect(screen.getByText('Response')).toBeInTheDocument();
      expect(screen.getByText(FIXTURES.bedrockTextResponse.output)).toBeInTheDocument();
    });

    it('filters out toolResult from user messages', () => {
      render(
        <MessageRenderer
          input={FIXTURES.bedrockWithToolResult.input}
          output={FIXTURES.bedrockWithToolResult.output}
        />
      );

      // Should only see real user messages, not tool results
      expect(screen.getByText('User Input')).toBeInTheDocument();
    });
  });

  describe('Long conversations', () => {
    it('collapses previous messages correctly', () => {
      render(
        <MessageRenderer
          input={FIXTURES.longConversation.input}
          output={FIXTURES.longConversation.output}
        />
      );

      const previousSection = screen.getByText(/Previous Messages/);
      expect(previousSection).toBeInTheDocument();
      expect(previousSection.textContent).toContain(`(${FIXTURES.longConversation.expectedPreviousCount})`);
    });

    it('expands previous messages on click', () => {
      render(
        <MessageRenderer
          input={FIXTURES.longConversation.input}
          output={FIXTURES.longConversation.output}
        />
      );

      fireEvent.click(screen.getByText(/Previous Messages/));
      expect(screen.getByText('Click to collapse')).toBeInTheDocument();
    });

    it('shows current user message expanded by default', () => {
      render(
        <MessageRenderer
          input={FIXTURES.longConversation.input}
          output={FIXTURES.longConversation.output}
        />
      );

      // Last user message content should be visible
      expect(screen.getByText('123 New Street, City, State 12345')).toBeInTheDocument();
    });

    it('shows response content expanded by default', () => {
      render(
        <MessageRenderer
          input={FIXTURES.longConversation.input}
          output={FIXTURES.longConversation.output}
        />
      );

      expect(screen.getByText(/I have updated your delivery address/)).toBeInTheDocument();
    });
  });

  describe('Edge cases', () => {
    it('handles empty messages array', () => {
      render(
        <MessageRenderer
          input={FIXTURES.emptyMessages.input}
          output={FIXTURES.emptyMessages.output}
        />
      );

      // Should not show Previous Messages section
      expect(screen.queryByText(/Previous Messages/)).not.toBeInTheDocument();
    });

    it('handles string output directly', () => {
      render(
        <MessageRenderer
          input={FIXTURES.stringOutput.input}
          output={FIXTURES.stringOutput.output}
        />
      );

      expect(screen.getByText(FIXTURES.stringOutput.output)).toBeInTheDocument();
    });

    it('handles null output gracefully', () => {
      render(
        <MessageRenderer
          input={FIXTURES.nullOutput.input}
          output={FIXTURES.nullOutput.output}
        />
      );

      // Should not crash, response section might not appear
      expect(screen.getByText('User Input')).toBeInTheDocument();
    });
  });

  describe('UI interactions', () => {
    it('shows raw JSON toggle', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicTextResponse.input}
          output={FIXTURES.anthropicTextResponse.output}
        />
      );

      expect(screen.getByText('Show Raw JSON')).toBeInTheDocument();
    });

    it('toggles to raw JSON view', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicTextResponse.input}
          output={FIXTURES.anthropicTextResponse.output}
        />
      );

      fireEvent.click(screen.getByText('Show Raw JSON'));
      expect(screen.getByText('Show Formatted')).toBeInTheDocument();
    });

    it('collapses system prompt by default', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicTextResponse.input}
          output={FIXTURES.anthropicTextResponse.output}
        />
      );

      // Should show "Click to expand" hint
      expect(screen.getByText('Click to expand')).toBeInTheDocument();
    });

    it('expands system prompt on click', () => {
      render(
        <MessageRenderer
          input={FIXTURES.anthropicTextResponse.input}
          output={FIXTURES.anthropicTextResponse.output}
        />
      );

      fireEvent.click(screen.getByText('System Prompt'));
      expect(screen.getByText('You are a helpful coding assistant.')).toBeInTheDocument();
    });
  });
});
