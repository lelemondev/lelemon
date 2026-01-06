'use client';

import { useState, useRef, useEffect, useCallback } from 'react';
import { ProviderSelect, type Provider } from './provider-select';
import { MessageList, type Message } from './message-list';
import { Send, Loader2, RefreshCw } from 'lucide-react';

// Generate a short session ID
function generateSessionId(): string {
  return `session-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

export function ChatInterface() {
  const [provider, setProvider] = useState<Provider>('bedrock');
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Initialize session on mount
  useEffect(() => {
    setSessionId(generateSessionId());
    inputRef.current?.focus();
  }, []);

  // Start new session
  const startNewSession = useCallback(() => {
    setSessionId(generateSessionId());
    setMessages([]);
  }, []);

  async function sendMessage(e: React.FormEvent) {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const userMessage = input.trim();
    setInput('');
    setIsLoading(true);

    // Add user message immediately
    setMessages(prev => [...prev, {
      role: 'user',
      content: userMessage
    }]);

    try {
      const res = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, message: userMessage, sessionId })
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || 'Failed to get response');
      }

      setMessages(prev => [...prev, {
        role: 'assistant',
        content: data.response,
        traceId: data.traceId,
        provider: data.provider,
        model: data.model,
        durationMs: data.durationMs,
        toolsUsed: data.toolsUsed,
      }]);
    } catch (error) {
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
        isError: true,
      }]);
    } finally {
      setIsLoading(false);
      inputRef.current?.focus();
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage(e);
    }
  }

  return (
    <div className="flex flex-col h-screen max-w-4xl mx-auto">
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4 border-b border-zinc-800">
        <div>
          <h1 className="text-xl font-semibold">Lelemon Playground</h1>
          <div className="flex items-center gap-2 mt-1">
            <span className="text-xs text-zinc-500">Session:</span>
            <code className="text-xs bg-zinc-800 px-2 py-0.5 rounded text-zinc-400">
              {sessionId || '...'}
            </code>
            <button
              onClick={startNewSession}
              className="text-xs text-zinc-500 hover:text-zinc-300 flex items-center gap-1 ml-2"
              title="Start new session"
            >
              <RefreshCw className="w-3 h-3" />
              New
            </button>
          </div>
        </div>
        <ProviderSelect value={provider} onChange={setProvider} />
      </header>

      {/* Messages */}
      <div className="flex-1 overflow-auto">
        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-zinc-500">
            <div className="text-6xl mb-4">🍋</div>
            <p className="text-lg">Start a conversation to test the SDK</p>
            <p className="text-sm mt-2">
              Try: "Search the web for latest AI news" or "Query the products database"
            </p>
          </div>
        ) : (
          <MessageList messages={messages} />
        )}
      </div>

      {/* Input */}
      <form onSubmit={sendMessage} className="p-4 border-t border-zinc-800">
        <div className="flex gap-3 items-end">
          <div className="flex-1 relative">
            <textarea
              ref={inputRef}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type a message... (Shift+Enter for new line)"
              className="w-full px-4 py-3 bg-zinc-900 border border-zinc-700 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              rows={1}
              style={{ minHeight: '48px', maxHeight: '200px' }}
              disabled={isLoading}
            />
          </div>
          <button
            type="submit"
            disabled={isLoading || !input.trim()}
            className="px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? (
              <Loader2 className="w-5 h-5 animate-spin" />
            ) : (
              <Send className="w-5 h-5" />
            )}
          </button>
        </div>
        <p className="text-xs text-zinc-600 mt-2 text-center">
          Provider: <span className="text-zinc-400">{provider}</span> •
          Messages are sent to the LLM and traced with Lelemon SDK
        </p>
      </form>
    </div>
  );
}
