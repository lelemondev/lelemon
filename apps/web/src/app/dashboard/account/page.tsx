'use client';

import { useState, useEffect } from 'react';
import { createClient } from '@/lib/supabase/client';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

export default function AccountPage() {
  const [user, setUser] = useState<{ email: string; created_at: string } | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const supabase = createClient();

  useEffect(() => {
    const getUser = async () => {
      const { data: { user } } = await supabase.auth.getUser();
      if (user) {
        setUser({
          email: user.email || '',
          created_at: user.created_at,
        });
      }
      setIsLoading(false);
    };
    getUser();
  }, [supabase]);

  if (isLoading) {
    return (
      <div className="space-y-8 max-w-2xl">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Account</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            Manage your account settings.
          </p>
        </div>
        <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Account</h1>
        <p className="text-zinc-500 dark:text-zinc-400 mt-1">
          Manage your account settings.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-1">Email</p>
            <p className="text-zinc-900 dark:text-white">{user?.email}</p>
          </div>
          <div>
            <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-1">Member since</p>
            <p className="text-zinc-900 dark:text-white">
              {user?.created_at ? new Date(user.created_at).toLocaleDateString() : '-'}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card className="border-red-200 dark:border-red-500/20">
        <CardHeader>
          <CardTitle className="text-red-600 dark:text-red-400">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mb-4">
            Permanently delete your account and all associated data. This action cannot be undone.
          </p>
          <Button variant="destructive" disabled>
            Delete Account
          </Button>
          <p className="text-xs text-zinc-400 dark:text-zinc-500 mt-2">
            Contact support to delete your account.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
