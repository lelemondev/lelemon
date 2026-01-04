'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useProject } from '@/lib/project-context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function ProjectsPage() {
  const router = useRouter();
  const { projects, currentProject, setCurrentProject, refreshProjects, isLoading } = useProject();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newProjectName, setNewProjectName] = useState('');
  const [creating, setCreating] = useState(false);
  const [newApiKey, setNewApiKey] = useState<string | null>(null);
  const [apiKeyCopied, setApiKeyCopied] = useState(false);

  const handleCreateProject = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newProjectName.trim()) return;

    setCreating(true);
    try {
      const response = await fetch('/api/v1/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newProjectName }),
      });

      if (response.ok) {
        const data = await response.json();
        setNewProjectName('');
        setShowCreateForm(false);
        await refreshProjects();
        if (data.apiKey) {
          setNewApiKey(data.apiKey);
        }
      }
    } catch (error) {
      console.error('Failed to create project:', error);
    } finally {
      setCreating(false);
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
    router.push('/dashboard/config');
  };

  const handleSelectProject = (project: typeof projects[0]) => {
    setCurrentProject(project);
  };

  if (isLoading) {
    return (
      <div className="space-y-8">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Projects</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            Manage your projects.
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-32 bg-zinc-100 dark:bg-zinc-800 rounded-2xl animate-pulse" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">      {newApiKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <Card className="w-full max-w-lg mx-4 border-amber-500 shadow-2xl">
            <CardHeader className="bg-gradient-to-r from-amber-500/10 to-orange-500/10 dark:from-amber-500/5 dark:to-orange-500/5 border-b border-zinc-200 dark:border-zinc-800">
              <CardTitle className="flex items-center gap-2">
                <svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                </svg>
                Your API Key
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
                  This is the only time you will see this API key. Copy it and store it securely. If you lose it, you will need to rotate your key.
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
                I have saved my key, continue to setup
              </Button>
            </CardContent>
          </Card>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Projects</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            Manage your projects.
          </p>
        </div>
        <Button
          onClick={() => setShowCreateForm(true)}
          className="bg-amber-500 hover:bg-amber-600 text-zinc-900 font-medium"
        >
          <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          New Project
        </Button>
      </div>

      {showCreateForm && (
        <Card className="border-amber-200 dark:border-amber-500/20 bg-amber-50/50 dark:bg-amber-500/5">
          <CardHeader>
            <CardTitle className="text-lg">Create New Project</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreateProject} className="flex gap-3">
              <Input
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                placeholder="Project name"
                className="flex-1"
                autoFocus
              />
              <Button type="submit" disabled={creating || !newProjectName.trim()}>
                {creating ? 'Creating...' : 'Create'}
              </Button>
              <Button type="button" variant="outline" onClick={() => setShowCreateForm(false)}>
                Cancel
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {projects.length === 0 && !showCreateForm ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-500/10 flex items-center justify-center mb-4">
              <svg className="w-8 h-8 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">No projects yet</h3>
            <p className="text-zinc-500 dark:text-zinc-400 mb-4">Create your first project to start tracing.</p>
            <Button
              onClick={() => setShowCreateForm(true)}
              className="bg-amber-500 hover:bg-amber-600 text-zinc-900"
            >
              Create Project
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((project) => {
            const isSelected = currentProject?.id === project.id;
            return (
              <Card
                key={project.id}
                className={`cursor-pointer transition-all ${
                  isSelected
                    ? 'border-amber-500 ring-2 ring-amber-500/20'
                    : 'hover:border-zinc-300 dark:hover:border-zinc-600'
                }`}
                onClick={() => handleSelectProject(project)}
              >
                <CardHeader className="pb-3">
                  <CardTitle className="flex items-center justify-between">
                    <div className="flex items-center gap-2 text-lg">
                      <div className={`w-2 h-2 rounded-full ${isSelected ? 'bg-amber-500' : 'bg-zinc-300 dark:bg-zinc-600'}`} />
                      {project.name}
                    </div>
                    {isSelected && (
                      <span className="text-xs font-normal text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-500/10 px-2 py-1 rounded">
                        Active
                      </span>
                    )}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-xs text-zinc-400 dark:text-zinc-500">
                    Created {new Date(project.createdAt).toLocaleDateString()}
                  </p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
