'use client';

import { Suspense, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';
import { useProject } from '@/lib/project-context';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

const API_URL = process.env.NEXT_PUBLIC_API_URL || '';
// The MCP we are allowed to hand a consent token back to (anti open-redirect).
const MCP_URL = process.env.NEXT_PUBLIC_MCP_URL || 'https://lelemon-mcp-production.up.railway.app/mcp';

/** Only allow returning to the configured MCP origin — never an attacker-supplied host. */
function safeReturnUrl(raw: string | null): URL | null {
  if (!raw) return null;
  try {
    const url = new URL(raw);
    const allowed = new URL(MCP_URL);
    return url.origin === allowed.origin ? url : null;
  } catch {
    return null;
  }
}

function AuthorizeContent() {
  const searchParams = useSearchParams();
  const { user, isAuthenticated, isLoading: authLoading } = useAuth();
  const { projects, currentProject, isLoading: projectsLoading } = useProject();

  const returnUrl = useMemo(() => safeReturnUrl(searchParams.get('return')), [searchParams]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const projectId = selectedId ?? currentProject?.id ?? projects[0]?.id ?? null;

  const handleAuthorize = async () => {
    if (!projectId || !returnUrl) return;
    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch(`${API_URL}/api/v1/dashboard/mcp/consent`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId }),
      });
      if (!res.ok) {
        throw new Error(res.status === 403 ? 'You do not own this project.' : 'Could not authorize.');
      }
      const { consentToken } = (await res.json()) as { consentToken: string };
      returnUrl.searchParams.set('consent', consentToken);
      window.location.href = returnUrl.toString();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong.');
      setSubmitting(false);
    }
  };

  if (authLoading || projectsLoading) {
    return <p className="text-muted-foreground">Loading…</p>;
  }

  if (!isAuthenticated) {
    const next = encodeURIComponent(`/dashboard/mcp/authorize?return=${encodeURIComponent(searchParams.get('return') ?? '')}`);
    return (
      <Card className="max-w-md mx-auto">
        <CardHeader>
          <CardTitle>Sign in to continue</CardTitle>
          <CardDescription>You need to be signed in to connect an agent to Lelemon.</CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild>
            <a href={`/login?redirect=${next}`}>Sign in</a>
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (!returnUrl) {
    return (
      <Card className="max-w-md mx-auto border-destructive/40">
        <CardHeader>
          <CardTitle>Invalid authorization request</CardTitle>
          <CardDescription>
            This page is opened by your agent during the &quot;Connect Claude&quot; flow. Start the
            connection from your MCP client (e.g. <code>claude mcp add</code>).
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card className="max-w-md mx-auto">
      <CardHeader>
        <div className="text-3xl mb-2" aria-hidden>
          🍋🔗
        </div>
        <CardTitle>Connect your agent to Lelemon</CardTitle>
        <CardDescription>
          Signed in as <span className="font-medium text-foreground">{user?.email}</span>. Choose the
          project your agent may read (traces, sessions, cost &amp; analytics — read-only).
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          {projects.map((p) => {
            const active = p.id === projectId;
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => setSelectedId(p.id)}
                className={`w-full flex items-center justify-between rounded-lg border px-4 py-3 text-left transition-colors cursor-pointer ${
                  active
                    ? 'border-emerald-500 bg-emerald-500/10'
                    : 'border-border hover:bg-muted/50'
                }`}
              >
                <span className="font-medium">{p.name}</span>
                {active && (
                  <svg className="w-5 h-5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                  </svg>
                )}
              </button>
            );
          })}
          {projects.length === 0 && (
            <p className="text-sm text-muted-foreground">You have no projects yet. Create one first.</p>
          )}
        </div>

        {error && <p className="text-sm text-destructive">{error}</p>}

        <div className="flex gap-2 pt-2">
          <Button onClick={handleAuthorize} disabled={!projectId || submitting} className="flex-1">
            {submitting ? 'Authorizing…' : 'Authorize'}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          Authorizing lets the connected agent query this project on your behalf. You can revoke
          access anytime by rotating credentials.
        </p>
      </CardContent>
    </Card>
  );
}

export default function McpAuthorizePage() {
  return (
    <div className="py-10">
      <Suspense fallback={<p className="text-muted-foreground">Loading…</p>}>
        <AuthorizeContent />
      </Suspense>
    </div>
  );
}
