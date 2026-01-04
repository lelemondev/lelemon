'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface LLMDataViewerProps {
  data: unknown;
  label: string;
  maxHeight?: string;
}

interface CollapsibleSectionProps {
  title: string;
  badge?: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

function CollapsibleSection({ title, badge, defaultOpen = false, children }: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border-b border-zinc-200 dark:border-zinc-700 last:border-b-0">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center gap-2 py-2 px-1 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors text-left"
      >
        <svg
          className={cn('w-4 h-4 text-zinc-400 transition-transform', isOpen && 'rotate-90')}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
        <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">{title}</span>
        {badge && (
          <span className="text-xs px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-500">
            {badge}
          </span>
        )}
      </button>
      {isOpen && <div className="pb-3 px-1">{children}</div>}
    </div>
  );
}

function TextBlock({ text, maxLines }: { text: string; maxLines?: number }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const lines = text.split('\n');
  const shouldTruncate = maxLines && lines.length > maxLines && !isExpanded;
  const displayText = shouldTruncate ? lines.slice(0, maxLines).join('\n') + '...' : text;

  return (
    <div className="relative">
      <pre className="text-xs text-zinc-600 dark:text-zinc-400 whitespace-pre-wrap font-mono bg-zinc-50 dark:bg-zinc-900 p-2 rounded border border-zinc-200 dark:border-zinc-700">
        {displayText}
      </pre>
      {shouldTruncate && (
        <button
          onClick={() => setIsExpanded(true)}
          className="text-xs text-amber-600 dark:text-amber-400 hover:underline mt-1"
        >
          Show more ({lines.length - maxLines!} more lines)
        </button>
      )}
    </div>
  );
}

interface Tool {
  name: string;
  description?: string;
  input_schema?: Record<string, unknown>;
}

interface Message {
  role: string;
  content?: string | MessageContentBlock[];
}

interface MessageContentBlock {
  type: string;
  text?: string;
  id?: string;
  name?: string;
  input?: Record<string, unknown>;
  content?: string;
  tool_use_id?: string;
}

interface ToolCall {
  id?: string;
  name?: string;
  type?: string;
  function?: { name: string; arguments: string };
}

function ToolCard({ tool }: { tool: Tool }) {
  const [showSchema, setShowSchema] = useState(false);

  return (
    <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <svg className="w-4 h-4 text-blue-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z" />
            </svg>
            <code className="text-sm font-semibold text-zinc-900 dark:text-white">{tool.name}</code>
          </div>
          {tool.description && (
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1 line-clamp-2">
              {tool.description}
            </p>
          )}
        </div>
        {tool.input_schema && (
          <button
            onClick={() => setShowSchema(!showSchema)}
            className="text-xs text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
          >
            {showSchema ? 'Hide' : 'Schema'}
          </button>
        )}
      </div>
      {showSchema && tool.input_schema && (
        <pre className="mt-2 text-xs bg-zinc-50 dark:bg-zinc-800 p-2 rounded overflow-auto max-h-32 text-zinc-600 dark:text-zinc-400">
          {JSON.stringify(tool.input_schema, null, 2)}
        </pre>
      )}
    </div>
  );
}

// Check if a message is a tool interaction (tool_use or tool_result)
function isToolInteraction(message: Message): boolean {
  const content = message.content;
  if (!Array.isArray(content) || content.length === 0) return false;
  return content[0].type === 'tool_use' || content[0].type === 'tool_result';
}

// Filter messages to get only real conversation messages
function filterRealMessages(messages: Message[]): Message[] {
  return messages.filter(msg => !isToolInteraction(msg));
}

// Extract tool interactions from messages
function extractToolInteractions(messages: Message[]): Message[] {
  return messages.filter(msg => isToolInteraction(msg));
}

function ToolInteractionCard({ message }: { message: Message }) {
  const content = message.content as MessageContentBlock[];
  const isToolUse = content[0].type === 'tool_use';

  if (isToolUse) {
    return (
      <div className="border border-blue-200 dark:border-blue-800 rounded-lg overflow-hidden">
        <div className="bg-blue-50 dark:bg-blue-900/30 px-3 py-2 flex items-center gap-2 border-b border-blue-200 dark:border-blue-800">
          <svg className="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z" />
          </svg>
          <span className="text-sm font-medium text-blue-700 dark:text-blue-300">Tool Call</span>
          <code className="text-sm font-semibold text-blue-600 dark:text-blue-400 ml-1">{content[0].name}</code>
        </div>
        <div className="p-3 bg-white dark:bg-zinc-900">
          {content[0].input && (
            <pre className="text-xs overflow-auto max-h-32 text-zinc-600 dark:text-zinc-400">
              {JSON.stringify(content[0].input, null, 2)}
            </pre>
          )}
        </div>
      </div>
    );
  }

  // Tool result
  return (
    <div className="border border-emerald-200 dark:border-emerald-800 rounded-lg overflow-hidden">
      <div className="bg-emerald-50 dark:bg-emerald-900/30 px-3 py-2 flex items-center gap-2 border-b border-emerald-200 dark:border-emerald-800">
        <svg className="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span className="text-sm font-medium text-emerald-700 dark:text-emerald-300">Tool Result</span>
      </div>
      <div className="p-3 bg-white dark:bg-zinc-900">
        <div className="text-sm whitespace-pre-wrap break-words text-zinc-700 dark:text-zinc-300">
          {typeof content[0].content === 'string'
            ? content[0].content
            : JSON.stringify(content[0].content, null, 2)}
        </div>
      </div>
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const content = message.content;
  const isUser = message.role === 'user';
  const isSystem = message.role === 'system';
  const textContent = typeof content === 'string' ? content : JSON.stringify(content);

  return (
    <div className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[85%] rounded-lg px-3 py-2 text-sm',
          isUser && 'bg-amber-500 text-zinc-900',
          !isUser && !isSystem && 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-white',
          isSystem && 'bg-purple-100 dark:bg-purple-900/30 text-purple-900 dark:text-purple-200 w-full max-w-full'
        )}
      >
        <div className="text-xs font-medium opacity-70 mb-1">{message.role}</div>
        <div className="whitespace-pre-wrap break-words">
          {textContent.length > 500 ? textContent.slice(0, 500) + '...' : textContent}
        </div>
      </div>
    </div>
  );
}

function ToolCallCard({ toolCall }: { toolCall: ToolCall }) {
  const [showArgs, setShowArgs] = useState(false);
  const name = toolCall.name || toolCall.function?.name || 'unknown';
  const args = toolCall.function?.arguments;

  return (
    <div className="border border-blue-200 dark:border-blue-800 rounded-lg p-3 bg-blue-50 dark:bg-blue-900/20">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z" />
          </svg>
          <code className="text-sm font-semibold text-blue-700 dark:text-blue-300">{name}</code>
        </div>
        {args && (
          <button
            onClick={() => setShowArgs(!showArgs)}
            className="text-xs text-blue-500 hover:underline"
          >
            {showArgs ? 'Hide args' : 'Show args'}
          </button>
        )}
      </div>
      {showArgs && args && (
        <pre className="mt-2 text-xs bg-white dark:bg-zinc-900 p-2 rounded overflow-auto max-h-32 text-zinc-600 dark:text-zinc-400 border border-blue-200 dark:border-blue-800">
          {(() => {
            try {
              return JSON.stringify(JSON.parse(args), null, 2);
            } catch {
              return args;
            }
          })()}
        </pre>
      )}
    </div>
  );
}

export function LLMDataViewer({ data, label, maxHeight = 'max-h-96' }: LLMDataViewerProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  if (data === null || data === undefined) {
    return (
      <div className="text-sm text-zinc-400 dark:text-zinc-500 italic">
        No {label.toLowerCase()}
      </div>
    );
  }

  // Try to parse if string
  let parsed: unknown = data;
  if (typeof data === 'string') {
    try {
      parsed = JSON.parse(data);
    } catch {
      // Keep as string
    }
  }

  // Check for known LLM fields
  const obj = typeof parsed === 'object' && parsed !== null ? (parsed as Record<string, unknown>) : null;
  const hasKnownFields = obj && (
    typeof obj.system === 'string' ||
    Array.isArray(obj.tools) ||
    Array.isArray(obj.messages) ||
    obj.content !== undefined ||
    Array.isArray(obj.tool_calls)
  );

  const renderSmartView = () => {
    if (!obj) {
      return typeof parsed === 'string' ? (
        <TextBlock text={parsed} maxLines={10} />
      ) : (
        <pre className="text-xs text-zinc-600 dark:text-zinc-400 whitespace-pre-wrap font-mono">
          {JSON.stringify(parsed, null, 2)}
        </pre>
      );
    }

    const systemPrompt = typeof obj.system === 'string' ? obj.system : null;
    const tools = Array.isArray(obj.tools) ? (obj.tools as Tool[]) : null;
    const messages = Array.isArray(obj.messages) ? (obj.messages as Message[]) : null;
    const toolCalls = Array.isArray(obj.tool_calls) ? (obj.tool_calls as ToolCall[]) : null;
    const content = obj.content;

    return (
      <div className="space-y-0">
        {systemPrompt && (
          <CollapsibleSection title="System Prompt" defaultOpen>
            <TextBlock text={systemPrompt} maxLines={8} />
          </CollapsibleSection>
        )}

        {tools && tools.length > 0 && (
          <CollapsibleSection title="Tools" badge={String(tools.length)}>
            <div className="space-y-2">
              {tools.map((tool, i) => (
                <ToolCard key={tool.name || i} tool={tool} />
              ))}
            </div>
          </CollapsibleSection>
        )}

        {messages && messages.length > 0 && (() => {
          const realMessages = filterRealMessages(messages);
          const toolInteractions = extractToolInteractions(messages);

          return (
            <>
              {realMessages.length > 0 && (
                <CollapsibleSection title="Messages" badge={String(realMessages.length)} defaultOpen>
                  <div className="space-y-2">
                    {realMessages.map((msg, i) => (
                      <MessageBubble key={i} message={msg} />
                    ))}
                  </div>
                </CollapsibleSection>
              )}

              {toolInteractions.length > 0 && (
                <CollapsibleSection title="Tool Interactions" badge={String(toolInteractions.length)} defaultOpen>
                  <div className="space-y-2">
                    {toolInteractions.map((msg, i) => (
                      <ToolInteractionCard key={i} message={msg} />
                    ))}
                  </div>
                </CollapsibleSection>
              )}
            </>
          );
        })()}

        {content !== undefined && (
          <CollapsibleSection title="Content" defaultOpen>
            {typeof content === 'string' ? (
              <TextBlock text={content} maxLines={10} />
            ) : (
              <pre className="text-xs text-zinc-600 dark:text-zinc-400 whitespace-pre-wrap font-mono bg-zinc-50 dark:bg-zinc-900 p-2 rounded">
                {JSON.stringify(content, null, 2)}
              </pre>
            )}
          </CollapsibleSection>
        )}

        {toolCalls && toolCalls.length > 0 && (
          <CollapsibleSection title="Tool Calls" badge={String(toolCalls.length)} defaultOpen>
            <div className="space-y-2">
              {toolCalls.map((tc, i) => (
                <ToolCallCard key={tc.id || i} toolCall={tc} />
              ))}
            </div>
          </CollapsibleSection>
        )}

        {/* Show remaining fields */}
        {(() => {
          const knownKeys = ['system', 'tools', 'messages', 'content', 'tool_calls'];
          const otherKeys = Object.keys(obj).filter(k => !knownKeys.includes(k));
          if (otherKeys.length === 0) return null;

          const otherData: Record<string, unknown> = {};
          otherKeys.forEach(k => { otherData[k] = obj[k]; });

          return (
            <CollapsibleSection title="Other Fields" badge={String(otherKeys.length)}>
              <pre className="text-xs text-zinc-600 dark:text-zinc-400 whitespace-pre-wrap font-mono bg-zinc-50 dark:bg-zinc-900 p-2 rounded overflow-auto max-h-32">
                {JSON.stringify(otherData, null, 2)}
              </pre>
            </CollapsibleSection>
          );
        })()}
      </div>
    );
  };

  return (
    <>
      <div className="relative">
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400">{label}</p>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={() => setIsModalOpen(true)}
          >
            <svg className="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
            </svg>
            Expand
          </Button>
        </div>
        <div className={cn('overflow-auto rounded-lg border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900', maxHeight)}>
          <div className="p-3">
            {hasKnownFields ? renderSmartView() : (
              <pre className="text-xs text-zinc-600 dark:text-zinc-400 whitespace-pre-wrap font-mono">
                {typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2)}
              </pre>
            )}
          </div>
        </div>
      </div>

      {/* Fullscreen Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="relative w-full max-w-4xl max-h-[90vh] bg-white dark:bg-zinc-900 rounded-xl shadow-2xl overflow-hidden flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold text-zinc-900 dark:text-white">{label}</h3>
              <Button variant="ghost" size="sm" onClick={() => setIsModalOpen(false)}>
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </Button>
            </div>
            <div className="flex-1 overflow-auto p-4">
              {hasKnownFields ? renderSmartView() : (
                <pre className="text-sm text-zinc-600 dark:text-zinc-400 whitespace-pre-wrap font-mono">
                  {typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2)}
                </pre>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
