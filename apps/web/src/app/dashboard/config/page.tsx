'use client';

import { Suspense, useState, useEffect } from 'react';
import Link from 'next/link';
import { useSearchParams, useRouter } from 'next/navigation';
import { useProject } from '@/lib/project-context';
import { dashboardAPI } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

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
        className="absolute top-2 right-2 p-2 rounded-md bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-white opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
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

function ConfigPageContent() {
  const { currentProject, isLoading, refreshProjects } = useProject();
  const searchParams = useSearchParams();
  const router = useRouter();

  // State for modals
  const [rotating, setRotating] = useState(false);
  const [newApiKey, setNewApiKey] = useState<string | null>(null);
  const [apiKeyCopied, setApiKeyCopied] = useState(false);
  const [welcomeKey, setWelcomeKey] = useState<string | null>(null);
  const [welcomeKeyCopied, setWelcomeKeyCopied] = useState(false);

  // State for project name editing
  const [projectName, setProjectName] = useState('');
  const [savingName, setSavingName] = useState(false);
  const [nameChanged, setNameChanged] = useState(false);

  // State for delete all traces
  const [showDeleteTracesModal, setShowDeleteTracesModal] = useState(false);
  const [deletingTraces, setDeletingTraces] = useState(false);
  const [deleteTracesConfirmText, setDeleteTracesConfirmText] = useState('');

  // Initialize project name
  useEffect(() => {
    if (currentProject) {
      setProjectName(currentProject.name);
    }
  }, [currentProject]);

  // Handle welcome flow for new users
  useEffect(() => {
    const isWelcome = searchParams.get('welcome') === 'true';
    const key = searchParams.get('key');

    if (isWelcome && key) {
      setWelcomeKey(key);
      refreshProjects();
      router.replace('/dashboard/config', { scroll: false });
    }
  }, [searchParams, refreshProjects, router]);

  const handleCopyWelcomeKey = async () => {
    if (!welcomeKey) return;
    await navigator.clipboard.writeText(welcomeKey);
    setWelcomeKeyCopied(true);
    setTimeout(() => setWelcomeKeyCopied(false), 2000);
  };

  const handleCloseWelcome = () => {
    setWelcomeKey(null);
    setWelcomeKeyCopied(false);
  };

  const handleRotateApiKey = async () => {
    if (!currentProject) return;
    setRotating(true);
    try {
      const result = await dashboardAPI.rotateProjectAPIKey(currentProject.id);
      setNewApiKey(result.apiKey);
      await refreshProjects();
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

  const handleSaveProjectName = async () => {
    if (!currentProject || !projectName.trim() || projectName === currentProject.name) return;
    setSavingName(true);
    try {
      await dashboardAPI.updateProject(currentProject.id, { name: projectName.trim() });
      await refreshProjects();
      setNameChanged(true);
      setTimeout(() => setNameChanged(false), 2000);
    } catch (error) {
      console.error('Failed to update project name:', error);
    } finally {
      setSavingName(false);
    }
  };

  const handleDeleteAllTraces = async () => {
    if (!currentProject || deleteTracesConfirmText !== 'delete all') return;
    setDeletingTraces(true);
    try {
      await dashboardAPI.deleteAllTraces(currentProject.id);
      setShowDeleteTracesModal(false);
      setDeleteTracesConfirmText('');
    } catch (error) {
      console.error('Failed to delete traces:', error);
    } finally {
      setDeletingTraces(false);
    }
  };

  const handleCloseDeleteTracesModal = () => {
    setShowDeleteTracesModal(false);
    setDeleteTracesConfirmText('');
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-zinc-900 dark:text-white">Project Settings</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">Configure your project and get started with the SDK.</p>
        </div>
        <div className="grid gap-6 lg:grid-cols-2">
          <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
          <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
        </div>
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
          <Button className="bg-amber-500 hover:bg-amber-600 text-zinc-900 cursor-pointer">Go to Projects</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Welcome Modal for New Users */}
      {welcomeKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <Card className="w-full max-w-lg mx-4 border-amber-500 shadow-2xl">
            <CardHeader className="bg-gradient-to-r from-amber-500/10 to-orange-500/10 dark:from-amber-500/5 dark:to-orange-500/5 border-b border-zinc-200 dark:border-zinc-800">
              <CardTitle className="flex items-center gap-2 text-xl">
                <span className="text-2xl">🍋</span>
                Welcome to Lelemon!
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-5 pt-6">
              <p className="text-zinc-600 dark:text-zinc-400">
                Your project is ready. Copy your API key to start tracing your LLM calls.
              </p>
              <div className="p-4 bg-amber-50 dark:bg-amber-500/10 rounded-lg border border-amber-200 dark:border-amber-500/20">
                <div className="flex items-center gap-2 mb-2">
                  <svg className="w-4 h-4 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                  </svg>
                  <span className="text-sm font-medium text-amber-700 dark:text-amber-300">Save this key now!</span>
                </div>
                <p className="text-xs text-amber-600 dark:text-amber-400">
                  This is the only time you&apos;ll see this API key. Store it securely.
                </p>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">Your API Key</label>
                <div className="flex gap-2">
                  <Input value={welcomeKey} readOnly className="font-mono text-sm" />
                  <Button
                    onClick={handleCopyWelcomeKey}
                    className={`cursor-pointer ${welcomeKeyCopied ? 'bg-emerald-500 hover:bg-emerald-600' : 'bg-amber-500 hover:bg-amber-600 text-zinc-900'}`}
                  >
                    {welcomeKeyCopied ? 'Copied!' : 'Copy'}
                  </Button>
                </div>
              </div>
              <div className="pt-2 space-y-3">
                <p className="text-sm text-zinc-500 dark:text-zinc-400">Add it to your environment:</p>
                <div className="bg-zinc-950 text-zinc-100 p-3 rounded-lg font-mono text-sm overflow-x-auto">
                  LELEMON_API_KEY={welcomeKey}
                </div>
              </div>
              <Button onClick={handleCloseWelcome} className="w-full bg-amber-500 hover:bg-amber-600 text-zinc-900 cursor-pointer">
                Got it, let&apos;s go!
              </Button>
            </CardContent>
          </Card>
        </div>
      )}

      {/* New API Key Modal */}
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
                    className={`cursor-pointer ${apiKeyCopied ? 'bg-emerald-500 hover:bg-emerald-600' : 'bg-amber-500 hover:bg-amber-600 text-zinc-900'}`}
                  >
                    {apiKeyCopied ? 'Copied!' : 'Copy'}
                  </Button>
                </div>
              </div>
              <Button onClick={handleCloseApiKeyModal} className="w-full cursor-pointer" variant="outline">
                I have saved my key
              </Button>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Delete All Traces Confirmation Modal */}
      {showDeleteTracesModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <Card className="w-full max-w-lg mx-4 border-red-500 shadow-2xl">
            <CardHeader className="bg-gradient-to-r from-red-500/10 to-red-600/10 dark:from-red-500/5 dark:to-red-600/5 border-b border-zinc-200 dark:border-zinc-800">
              <CardTitle className="flex items-center gap-2 text-red-600 dark:text-red-400">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
                Delete All Traces
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 pt-6">
              <div className="p-4 bg-red-50 dark:bg-red-500/10 rounded-lg border border-red-200 dark:border-red-500/20">
                <p className="text-sm text-red-700 dark:text-red-300">
                  This action is <strong>irreversible</strong>. All trace data including spans, tokens, and cost information will be permanently deleted.
                </p>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Type <span className="font-mono bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">delete all</span> to confirm
                </label>
                <Input
                  value={deleteTracesConfirmText}
                  onChange={(e) => setDeleteTracesConfirmText(e.target.value)}
                  placeholder="delete all"
                  className="font-mono"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  onClick={handleCloseDeleteTracesModal}
                  variant="outline"
                  className="flex-1 cursor-pointer"
                  disabled={deletingTraces}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleDeleteAllTraces}
                  variant="destructive"
                  disabled={deleteTracesConfirmText !== 'delete all' || deletingTraces}
                  className="flex-1 cursor-pointer"
                >
                  {deletingTraces ? 'Deleting...' : 'Delete All Traces'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-zinc-900 dark:text-white">Project Settings</h1>
        <p className="text-zinc-500 dark:text-zinc-400 mt-1">Configure your project and get started with the SDK.</p>
      </div>

      {/* Main Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Left Column - Settings */}
        <div className="space-y-6">
          {/* Project Name */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
                </svg>
                Project Name
              </CardTitle>
              <CardDescription>Change your project&apos;s display name.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Input
                  value={projectName}
                  onChange={(e) => setProjectName(e.target.value)}
                  placeholder="My Project"
                  className="flex-1"
                />
                <Button
                  onClick={handleSaveProjectName}
                  disabled={savingName || !projectName.trim() || projectName === currentProject.name}
                  className={`cursor-pointer ${nameChanged ? 'bg-emerald-500 hover:bg-emerald-600' : 'bg-amber-500 hover:bg-amber-600 text-zinc-900'}`}
                >
                  {nameChanged ? 'Saved!' : savingName ? 'Saving...' : 'Save'}
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* API Key */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                </svg>
                API Key
              </CardTitle>
              <CardDescription>Your API key is shown only once. Rotate if you lost it.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Input
                  value={currentProject.apiKeyHint || '•'.repeat(32)}
                  readOnly
                  className="font-mono text-sm flex-1"
                />
                <Button
                  onClick={handleRotateApiKey}
                  disabled={rotating}
                  className="bg-amber-500 hover:bg-amber-600 text-zinc-900 cursor-pointer"
                >
                  {rotating ? 'Rotating...' : 'Rotate'}
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Danger Zone */}
          <Card className="border-red-200 dark:border-red-500/20">
            <CardHeader>
              <CardTitle className="text-red-600 dark:text-red-400">Danger Zone</CardTitle>
              <CardDescription>Irreversible actions for this project.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="font-medium text-zinc-900 dark:text-white">Delete all traces</p>
                  <p className="text-sm text-zinc-500 dark:text-zinc-400">Permanently delete all trace data.</p>
                </div>
                <Button
                  variant="destructive"
                  onClick={() => setShowDeleteTracesModal(true)}
                  className="cursor-pointer"
                >
                  Delete All
                </Button>
              </div>
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="font-medium text-zinc-900 dark:text-white">Delete project</p>
                  <p className="text-sm text-zinc-500 dark:text-zinc-400">Delete project and all data.</p>
                </div>
                <Button variant="destructive" disabled className="cursor-not-allowed">Delete</Button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Column - Quick Start */}
        <div>
          <Card className="overflow-hidden h-fit sticky top-6">
            <CardHeader className="bg-gradient-to-r from-amber-500/10 to-orange-500/10 dark:from-amber-500/5 dark:to-orange-500/5 border-b border-zinc-200 dark:border-zinc-800">
              <CardTitle className="flex items-center gap-2">
                <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
                </svg>
                Quick Start Guide
              </CardTitle>
              <CardDescription>Get up and running in 3 simple steps.</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              {/* Step 1 */}
              <div className="p-5 border-b border-zinc-200 dark:border-zinc-800">
                <div className="flex items-center gap-3 mb-3">
                  <span className="flex items-center justify-center w-7 h-7 rounded-full bg-amber-500 text-zinc-900 text-sm font-bold">1</span>
                  <span className="font-medium text-zinc-900 dark:text-white">Install the SDK</span>
                </div>
                <CodeBlock code="npm install @lelemondev/sdk" />
              </div>

              {/* Step 2 */}
              <div className="p-5 border-b border-zinc-200 dark:border-zinc-800">
                <div className="flex items-center gap-3 mb-3">
                  <span className="flex items-center justify-center w-7 h-7 rounded-full bg-amber-500 text-zinc-900 text-sm font-bold">2</span>
                  <span className="font-medium text-zinc-900 dark:text-white">Initialize once at startup</span>
                </div>
                <CodeBlock code={`import { init } from '@lelemondev/sdk';

init({ apiKey: process.env.LELEMON_API_KEY });`} />
              </div>

              {/* Step 3 */}
              <div className="p-5">
                <div className="flex items-center gap-3 mb-3">
                  <span className="flex items-center justify-center w-7 h-7 rounded-full bg-amber-500 text-zinc-900 text-sm font-bold">3</span>
                  <span className="font-medium text-zinc-900 dark:text-white">Trace your LLM calls</span>
                </div>
                <CodeBlock code={`import { trace, flush } from '@lelemondev/sdk';

const t = trace({ input: userMessage });

try {
  const result = await myAgent(userMessage);
  t.success(result.messages);
} catch (error) {
  t.error(error);
  throw error;
}

// Serverless: flush before response
await flush();`} />
              </div>

              {/* Docs Link */}
              <div className="p-5 bg-zinc-50 dark:bg-zinc-900/50 border-t border-zinc-200 dark:border-zinc-800">
                <a
                  href="https://lelemondev.github.io/lelemondev-sdk/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-center gap-2 text-sm font-medium text-amber-600 dark:text-amber-400 hover:underline"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
                  </svg>
                  View Full Documentation
                </a>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

function ConfigPageFallback() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-zinc-900 dark:text-white">Project Settings</h1>
        <p className="text-zinc-500 dark:text-zinc-400 mt-1">Configure your project and get started with the SDK.</p>
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
        <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
      </div>
    </div>
  );
}

export default function ConfigPage() {
  return (
    <Suspense fallback={<ConfigPageFallback />}>
      <ConfigPageContent />
    </Suspense>
  );
}
