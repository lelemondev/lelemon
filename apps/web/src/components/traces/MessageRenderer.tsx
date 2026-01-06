'use client';

import { useState, useMemo } from 'react';
import { cn } from '@/lib/utils';

interface MessageRendererProps {
  input: unknown;
  output: unknown;
}

// Types for different message formats
interface BedrockToolUse {
  toolUse: {
    name: string;
    input?: Record<string, unknown>;
    toolUseId?: string;
  };
}

interface BedrockToolResult {
  toolResult: {
    toolUseId: string;
    status: string;
    content: unknown;
  };
}

interface ContentBlock {
  type?: string;
  text?: string;
  id?: string;
  name?: string;
  input?: Record<string, unknown>;
  content?: string;
  tool_use_id?: string;
}

interface ChatMessage {
  role: string;
  content: string | (ContentBlock | BedrockToolUse | BedrockToolResult)[];
  name?: string;
}

// Represents a single event in the conversation timeline
// Note: tool-call and tool-result are now shown as separate spans in the tree
interface TimelineEvent {
  type: 'user' | 'assistant-text' | 'response';
  content: string;
}

export function MessageRenderer({ input, output }: MessageRendererProps) {
  const [showRawJson, setShowRawJson] = useState(false);
  const [systemCollapsed, setSystemCollapsed] = useState(true);

  const parsedData = useMemo(() => {
    let parsed = input;
    if (typeof input === 'string') {
      try {
        parsed = JSON.parse(input);
      } catch {
        return null;
      }
    }

    const obj = typeof parsed === 'object' && parsed !== null
      ? (parsed as Record<string, unknown>)
      : null;

    if (!obj) return null;

    // Handle system prompt - can be string, array of {text}, or object
    let system: string | null = null;
    if (typeof obj.system === 'string') {
      system = obj.system;
    } else if (Array.isArray(obj.system)) {
      system = obj.system
        .map((s: { text?: string }) => s.text || '')
        .filter(Boolean)
        .join('\n');
    }

    return {
      system,
      messages: Array.isArray(obj.messages) ? (obj.messages as ChatMessage[]) : [],
      tools: Array.isArray(obj.tools) ? obj.tools : null,
    };
  }, [input]);

  // Parse output content
  const outputContent = useMemo((): string | null => {
    if (!output) return null;

    // If it's already a string
    if (typeof output === 'string') {
      // Try to parse as JSON in case it's a stringified response
      try {
        const parsed = JSON.parse(output);
        if (typeof parsed === 'string') return parsed;
        if (parsed.content) {
          if (typeof parsed.content === 'string') return parsed.content;
          if (Array.isArray(parsed.content)) {
            return parsed.content
              .filter((b: ContentBlock) => b.text || b.type === 'text')
              .map((b: ContentBlock) => b.text)
              .join('\n');
          }
        }
        return JSON.stringify(parsed, null, 2);
      } catch {
        return output;
      }
    }

    const obj = output as Record<string, unknown>;
    if (typeof obj.content === 'string') {
      return obj.content;
    }
    if (Array.isArray(obj.content)) {
      const texts = obj.content
        .filter((b: ContentBlock) => b.text && (b.type === 'text' || !b.type))
        .map((b: ContentBlock) => b.text);
      if (texts.length > 0) return texts.join('\n');
    }

    return JSON.stringify(output, null, 2);
  }, [output]);

  // Build timeline from messages (excluding tool calls/results - they are now spans)
  const timeline = useMemo((): TimelineEvent[] => {
    if (!parsedData) return [];

    const events: TimelineEvent[] = [];

    for (const msg of parsedData.messages) {
      const content = msg.content;

      if (msg.role === 'user') {
        // Skip tool result messages - they are now shown as spans
        if (Array.isArray(content)) {
          const hasToolResult = content.some((block) => {
            const bedrockResult = block as BedrockToolResult;
            return bedrockResult.toolResult !== undefined;
          });
          if (!hasToolResult) {
            // Regular user message
            const text = extractText(content);
            if (text) {
              events.push({ type: 'user', content: text });
            }
          }
        } else if (typeof content === 'string') {
          events.push({ type: 'user', content });
        }
      } else if (msg.role === 'assistant') {
        if (Array.isArray(content)) {
          // Extract text part only (tool calls are now spans)
          const text = extractText(content);
          if (text) {
            events.push({ type: 'assistant-text', content: text });
          }
        } else if (typeof content === 'string') {
          events.push({ type: 'assistant-text', content });
        }
      }
    }

    return events;
  }, [parsedData]);

  function extractText(content: (ContentBlock | BedrockToolUse | BedrockToolResult)[]): string {
    return content
      .filter((b): b is ContentBlock => {
        const block = b as ContentBlock;
        return Boolean(block.text && (block.type === 'text' || !block.type));
      })
      .map(b => b.text)
      .join('\n');
  }

  if (!parsedData) {
    return (
      <div className="space-y-4">
        <pre className="text-xs bg-zinc-50 dark:bg-zinc-800 p-4 rounded-lg overflow-auto max-h-64">
          {JSON.stringify(input, null, 2)}
        </pre>
        {output !== null && output !== undefined && (
          <pre className="text-xs bg-zinc-50 dark:bg-zinc-800 p-4 rounded-lg overflow-auto max-h-64">
            {JSON.stringify(output, null, 2)}
          </pre>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Toggle Raw JSON */}
      <div className="flex justify-end">
        <button
          onClick={() => setShowRawJson(!showRawJson)}
          className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 underline"
        >
          {showRawJson ? 'Show Formatted' : 'Show Raw JSON'}
        </button>
      </div>

      {showRawJson ? (
        <pre className="text-xs bg-zinc-50 dark:bg-zinc-800 p-4 rounded-lg overflow-auto max-h-96">
          {JSON.stringify({ input, output }, null, 2)}
        </pre>
      ) : (
        <div className="space-y-3">
          {/* System Prompt - Collapsible */}
          {parsedData.system && (
            <div className="rounded-lg border border-purple-200 dark:border-purple-800 bg-purple-50 dark:bg-purple-900/20 overflow-hidden">
              <button
                onClick={() => setSystemCollapsed(!systemCollapsed)}
                className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-purple-100 dark:hover:bg-purple-900/30 transition-colors"
              >
                <svg
                  className={cn(
                    'w-4 h-4 text-purple-600 dark:text-purple-400 transition-transform',
                    !systemCollapsed && 'rotate-90'
                  )}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                </svg>
                <svg
                  className="w-4 h-4 text-purple-600 dark:text-purple-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" />
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span className="text-xs font-medium text-purple-700 dark:text-purple-300 uppercase">
                  System Prompt
                </span>
                <span className="text-xs text-purple-500 dark:text-purple-400 ml-auto">
                  {systemCollapsed ? 'Click to expand' : 'Click to collapse'}
                </span>
              </button>
              {!systemCollapsed && (
                <div className="px-3 pb-3 border-t border-purple-200 dark:border-purple-800">
                  <pre className="text-xs text-purple-800 dark:text-purple-200 whitespace-pre-wrap mt-2 max-h-48 overflow-auto">
                    {parsedData.system}
                  </pre>
                </div>
              )}
            </div>
          )}

          {/* Conversation Messages */}
          <div className="space-y-3">
            {timeline.map((event, i) => (
              <MessageCard key={i} event={event} />
            ))}

            {/* Final Response */}
            {outputContent && (
              <MessageCard event={{ type: 'response', content: outputContent }} />
            )}
          </div>

          {/* Available Tools */}
          {parsedData.tools && parsedData.tools.length > 0 && (
            <div className="mt-4 pt-4 border-t border-zinc-200 dark:border-zinc-700">
              <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-2">
                Available Tools ({parsedData.tools.length})
              </p>
              <div className="flex flex-wrap gap-2">
                {parsedData.tools.map((tool: { name: string }, i: number) => (
                  <div
                    key={i}
                    className="text-xs bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 px-2 py-1 rounded border border-blue-200 dark:border-blue-800"
                  >
                    {tool.name}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

const MESSAGE_STYLES: Record<TimelineEvent['type'], {
  bg: string;
  border: string;
  text: string;
  label: string;
  labelColor: string;
  icon: string;
}> = {
  user: {
    bg: 'bg-amber-50 dark:bg-amber-900/20',
    border: 'border-amber-200 dark:border-amber-800',
    text: 'text-amber-900 dark:text-amber-100',
    label: 'User Input',
    labelColor: 'text-amber-600 dark:text-amber-400',
    icon: 'M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z',
  },
  'assistant-text': {
    bg: 'bg-zinc-50 dark:bg-zinc-800/50',
    border: 'border-zinc-200 dark:border-zinc-700',
    text: 'text-zinc-800 dark:text-zinc-200',
    label: 'Assistant',
    labelColor: 'text-zinc-600 dark:text-zinc-400',
    icon: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z',
  },
  response: {
    bg: 'bg-emerald-50 dark:bg-emerald-900/20',
    border: 'border-emerald-200 dark:border-emerald-800',
    text: 'text-emerald-900 dark:text-emerald-100',
    label: 'Response',
    labelColor: 'text-emerald-600 dark:text-emerald-400',
    icon: 'M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
  },
};

function MessageCard({ event }: { event: TimelineEvent }) {
  const style = MESSAGE_STYLES[event.type];
  const [expanded, setExpanded] = useState(true);

  return (
    <div className={cn('rounded-lg border overflow-hidden', style.bg, style.border)}>
      {/* Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
      >
        <svg
          className={cn('w-3 h-3 text-zinc-400 transition-transform flex-shrink-0', expanded && 'rotate-90')}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
        <svg
          className={cn('w-4 h-4 flex-shrink-0', style.labelColor)}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={1.5}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d={style.icon} />
        </svg>
        <span className={cn('text-xs font-semibold uppercase', style.labelColor)}>
          {style.label}
        </span>
      </button>

      {/* Content */}
      {expanded && (
        <div className={cn('px-4 pb-4 border-t', style.border)}>
          {event.content && (
            <div className={cn('text-sm mt-3 whitespace-pre-wrap leading-relaxed', style.text)}>
              {event.content.length > 2000 ? event.content.slice(0, 2000) + '...' : event.content}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
