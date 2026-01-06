'use client';

import { TraceInfo } from './trace-info';
import { User, Bot, AlertCircle, Wrench } from 'lucide-react';

export interface Message {
  role: 'user' | 'assistant';
  content: string;
  traceId?: string;
  provider?: string;
  model?: string;
  durationMs?: number;
  toolsUsed?: string[];
  isError?: boolean;
}

interface MessageListProps {
  messages: Message[];
}

export function MessageList({ messages }: MessageListProps) {
  return (
    <div className="p-4 space-y-4">
      {messages.map((message, index) => (
        <MessageItem key={index} message={message} />
      ))}
    </div>
  );
}

function MessageItem({ message }: { message: Message }) {
  const isUser = message.role === 'user';

  return (
    <div className={`flex gap-3 ${isUser ? 'justify-end' : 'justify-start'}`}>
      {!isUser && (
        <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${
          message.isError ? 'bg-red-500/20' : 'bg-blue-500/20'
        }`}>
          {message.isError ? (
            <AlertCircle className="w-4 h-4 text-red-400" />
          ) : (
            <Bot className="w-4 h-4 text-blue-400" />
          )}
        </div>
      )}

      <div className={`max-w-[80%] ${isUser ? 'order-first' : ''}`}>
        <div className={`px-4 py-3 rounded-2xl ${
          isUser
            ? 'bg-blue-600 text-white rounded-br-md'
            : message.isError
              ? 'bg-red-500/10 border border-red-500/20 text-red-300 rounded-bl-md'
              : 'bg-zinc-800 text-zinc-100 rounded-bl-md'
        }`}>
          <p className="whitespace-pre-wrap">{message.content}</p>
        </div>

        {/* Metadata for assistant messages */}
        {!isUser && !message.isError && (
          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
            {message.provider && (
              <span className="px-2 py-0.5 bg-zinc-800 rounded">
                {message.provider}
              </span>
            )}
            {message.model && (
              <span className="px-2 py-0.5 bg-zinc-800 rounded">
                {message.model}
              </span>
            )}
            {message.durationMs && (
              <span>{(message.durationMs / 1000).toFixed(2)}s</span>
            )}
            {message.toolsUsed && message.toolsUsed.length > 0 && (
              <span className="flex items-center gap-1 px-2 py-0.5 bg-amber-500/10 text-amber-400 rounded">
                <Wrench className="w-3 h-3" />
                {message.toolsUsed.join(', ')}
              </span>
            )}
            {message.traceId && (
              <TraceInfo traceId={message.traceId} />
            )}
          </div>
        )}
      </div>

      {isUser && (
        <div className="w-8 h-8 rounded-full bg-zinc-700 flex items-center justify-center flex-shrink-0">
          <User className="w-4 h-4 text-zinc-300" />
        </div>
      )}
    </div>
  );
}
