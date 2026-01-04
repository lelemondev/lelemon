'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useProject } from '@/lib/project-context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

function CodeBlock({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      <pre className="bg-zinc-950 text-zinc-100 p-4 rounded-lg font-mono text-sm overflow-x-auto">
        <code>{code}</code>
      </pre>
      <button
        onClick={handleCopy}
        className="absolute top-2 right-2 p-2 rounded-md bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-white opacity-0 group-hover:opacity-100 transition-opacity"
      >
        {copied ? (
          <svg className="w-4 h-4 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        ) : (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
          </svg>
        )}
      </button>
    </div>
  );
}

export default function ConfigPage() {
  const { currentProject, isLoading } = useProject();
  const [rotating, setRotating] = useState(false);
  const [newApiKey, setNewApiKey] = useState<string | null>(null);
  const [apiKeyCopied, setApiKeyCopied] = useState(false);

  const handleRotateApiKey = async () => {
    if (!currentProject) return;
    setRotating(true);
    try {
      const response = await fetch('/api/v1/projects/api-key', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ projectId: currentProject.id }) });
      if (response.ok) {
        const data = await response.json();
        setNewApiKey(data.apiKey);
      }
    } catch (error) {
      console.error('Failed to rotate API key:', error);
    } finally {
      setRotating(false);
    }
  };

  const handleCopyApiKey = async () => {
    if (!newApiKey) return;
    await navigator.clipboard.writeText(newApiKey);
    setApiKeyCopied(true);
    setTimeout(() => setApiKeyCopied(false), 2000);
  };

  const handleCloseApiKeyModal = () => {
    setNewApiKey(null);
    setApiKeyCopied(false);
  };

  if (isLoading) {
    return (
      <div className="space-y-8 max-w-2xl">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Project Config</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            Configure your project settings.
          </p>
        </div>
        <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
      </div>
    );
  }

  if (!currentProject) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-500/10 flex items-center justify-center mb-4">
          <svg className="w-8 h-8 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">No project selected</h3>
        <p className="text-zinc-500 dark:text-zinc-400 mb-4">Select or create a project first.</p>
        <Link href="/dashboard/projects">
          <Button className="bg-amber-500 hover:bg-amber-600 text-zinc-900">
            Go to Projects
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-2xl">
      {newApiKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <Card className="w-full max-w-lg mx-4 border-amber-500 shadow-2xl">
            <CardHeader className="bg-gradient-to-r from-amber-500/10 to-orange-500/10 dark:from-amber-500/5 dark:to-orange-500/5 border-b border-zinc-200 dark:border-zinc-800">
              <CardTitle className="flex items-center gap-2">
                <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                </svg>
                New API Key
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 pt-6">
              <div className="p-4 bg-amber-50 dark:bg-amber-500/10 rounded-lg border border-amber-200 dark:border-amber-500/20">
                <div className="flex items-center gap-2 mb-2">
                  <svg className="w-4 h-4 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                  </svg>
                  <span className="text-sm font-medium text-amber-700 dark:text-amber-300">Save this key now!</span>
                </div>
                <p className="text-xs text-amber-600 dark:text-amber-400">
                  This is the only time you will see this API key. Your old key has been invalidated.
                </p>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">API Key</label>
                <div className="flex gap-2">
                  <Input value={newApiKey} readOnly className="font-mono text-sm" />
                  <Button
                    onClick={handleCopyApiKey}
                    className={apiKeyCopied ? 'bg-emerald-500 hover:bg-emerald-600' : 'bg-amber-500 hover:bg-amber-600 text-zinc-900'}
                  >
                    {apiKeyCopied ? 'Copied!' : 'Copy'}
                  </Button>
                </div>
              </div>
              <Button onClick={handleCloseApiKeyModal} className="w-full" variant="outline">
                I have saved my key
              </Button>
            </CardContent>
          </Card>
        </div>
      )}

      <div>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Project Config</h1>
        <p className="text-zinc-500 dark:text-zinc-400 mt-1">
          Configure settings for <span className="font-medium text-zinc-700 dark:text-zinc-300">{currentProject.name}</span>
        </p>
      </div>

      {/* API Key */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
            </svg>
            API Key
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            Your API key is shown only once when you create a project. If you lost it, you can rotate it to get a new one.
          </p>
          <div className="flex gap-2">
            <Input
              value={currentProject.apiKeyHint ? currentProject.apiKeyHint + '•'.repeat(20) : '•'.repeat(32)}
              readOnly
              className="font-mono text-sm"
            />
            <Button
              onClick={handleRotateApiKey}
              disabled={rotating}
              className="bg-amber-500 hover:bg-amber-600 text-zinc-900"
            >
              {rotating ? 'Rotating...' : 'Rotate Key'}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Quick Start */}
      <Card className="overflow-hidden">
        <CardHeader className="bg-gradient-to-r from-amber-500/10 to-orange-500/10 dark:from-amber-500/5 dark:to-orange-500/5 border-b border-zinc-200 dark:border-zinc-800">
          <CardTitle className="flex items-center gap-2">
            <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
            </svg>
            Quick Start
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {/* Step 1 */}
          <div className="p-4 border-b border-zinc-200 dark:border-zinc-800">
            <div className="flex items-center gap-3 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-amber-500 text-zinc-900 text-xs font-bold">1</span>
              <span className="text-sm font-medium text-zinc-900 dark:text-white">Install the SDK</span>
            </div>
            <CodeBlock code="npm install @lelemondev/sdk" />
          </div>

          {/* Step 2 */}
          <div className="p-4 border-b border-zinc-200 dark:border-zinc-800">
            <div className="flex items-center gap-3 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-amber-500 text-zinc-900 text-xs font-bold">2</span>
              <span className="text-sm font-medium text-zinc-900 dark:text-white">Initialize once</span>
            </div>
            <CodeBlock code={`import { init } from '@lelemondev/sdk';

// Call once at app startup
init({ apiKey: process.env.LELEMON_API_KEY });`} />
          </div>

          {/* Step 3 */}
          <div className="p-4">
            <div className="flex items-center gap-3 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-amber-500 text-zinc-900 text-xs font-bold">3</span>
              <span className="text-sm font-medium text-zinc-900 dark:text-white">Trace your agent</span>
            </div>
            <CodeBlock code={`import { trace, flush } from '@lelemondev/sdk';

// Fire-and-forget - no awaits needed!
const t = trace({ input: userMessage });

try {
  const result = await myAgent(userMessage);
  t.success(result.messages);
} catch (error) {
  t.error(error);
  throw error;
}

// For serverless: flush before response
await flush();`} />
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-red-200 dark:border-red-500/20">
        <CardHeader>
          <CardTitle className="text-red-600 dark:text-red-400">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-zinc-900 dark:text-white">Delete all traces</p>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Permanently delete all trace data for this project.
              </p>
            </div>
            <Button variant="destructive" disabled>
              Delete All
            </Button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-zinc-900 dark:text-white">Delete project</p>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Delete this project and all associated data.
              </p>
            </div>
            <Button variant="destructive" disabled>
              Delete
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
