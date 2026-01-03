'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

export default function SettingsPage() {
  const [projectName, setProjectName] = useState('My Project');
  const [apiKey, setApiKey] = useState('le_sk_••••••••••••••••');
  const [showKey, setShowKey] = useState(false);

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">
          Manage your project configuration and API keys.
        </p>
      </div>

      {/* Project Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Project</CardTitle>
          <CardDescription>
            Basic project information and configuration.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Project Name</label>
            <Input
              value={projectName}
              onChange={(e) => setProjectName(e.target.value)}
              className="font-mono"
            />
          </div>
          <Button>Save Changes</Button>
        </CardContent>
      </Card>

      {/* API Key */}
      <Card>
        <CardHeader>
          <CardTitle>API Key</CardTitle>
          <CardDescription>
            Use this key to authenticate SDK requests. Keep it secret!
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Current API Key</label>
            <div className="flex gap-2">
              <Input
                value={showKey ? 'le_sk_abc123def456ghi789jkl0' : apiKey}
                readOnly
                className="font-mono"
              />
              <Button
                variant="outline"
                onClick={() => setShowKey(!showKey)}
              >
                {showKey ? 'Hide' : 'Show'}
              </Button>
            </div>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => navigator.clipboard.writeText('le_sk_abc123def456ghi789jkl0')}
            >
              Copy Key
            </Button>
            <Button variant="destructive">
              Rotate Key
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            Rotating the key will invalidate the current key immediately.
            All SDK instances will need to be updated with the new key.
          </p>
        </CardContent>
      </Card>

      {/* SDK Installation */}
      <Card>
        <CardHeader>
          <CardTitle>Quick Start</CardTitle>
          <CardDescription>
            Install the SDK and start tracing in minutes.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">1. Install the SDK</label>
            <pre className="bg-zinc-900 text-zinc-100 p-3 rounded-lg font-mono text-sm overflow-x-auto">
              npm install @lelemon/sdk
            </pre>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">2. Initialize the tracer</label>
            <pre className="bg-zinc-900 text-zinc-100 p-3 rounded-lg font-mono text-sm overflow-x-auto">
{`import { LLMTracer } from '@lelemon/sdk';

const tracer = new LLMTracer({
  apiKey: process.env.LELEMON_API_KEY,
});`}
            </pre>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">3. Start tracing</label>
            <pre className="bg-zinc-900 text-zinc-100 p-3 rounded-lg font-mono text-sm overflow-x-auto">
{`const trace = await tracer.startTrace({
  sessionId: 'session-123',
  userId: 'user-456',
});

const span = trace.startSpan({
  type: 'llm',
  name: 'chat-completion',
  input: { messages },
});

// ... your LLM call ...

span.end({
  output: response,
  model: 'gpt-4',
  inputTokens: usage.prompt_tokens,
  outputTokens: usage.completion_tokens,
});

await trace.end();`}
            </pre>
          </div>
        </CardContent>
      </Card>

      {/* Data Retention */}
      <Card>
        <CardHeader>
          <CardTitle>Data Retention</CardTitle>
          <CardDescription>
            Configure how long trace data is stored.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Retention Period</p>
              <p className="text-sm text-muted-foreground">
                Traces older than this will be automatically deleted.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Input
                type="number"
                defaultValue={30}
                className="w-20 font-mono"
              />
              <span className="text-sm text-muted-foreground">days</span>
            </div>
          </div>
          <Button variant="outline">Update Retention</Button>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-red-500/50">
        <CardHeader>
          <CardTitle className="text-red-500">Danger Zone</CardTitle>
          <CardDescription>
            Irreversible actions. Proceed with caution.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Delete All Traces</p>
              <p className="text-sm text-muted-foreground">
                Permanently delete all trace data for this project.
              </p>
            </div>
            <Button variant="destructive">Delete All</Button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Delete Project</p>
              <p className="text-sm text-muted-foreground">
                Delete this project and all associated data.
              </p>
            </div>
            <Button variant="destructive">Delete Project</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
