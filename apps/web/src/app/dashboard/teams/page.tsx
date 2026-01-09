'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/lib/auth-context';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { EnterpriseGate } from '@/ee/components/feature-gate';

interface TeamMember {
  id: string;
  userId: string;
  email: string;
  name: string;
  role: 'owner' | 'admin' | 'member' | 'viewer';
  joinedAt: string | null;
}

export default function TeamsPage() {
  const { user } = useAuth();
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // TODO: Fetch team members from API
    setIsLoading(false);
    setMembers([
      {
        id: '1',
        userId: user?.id || '',
        email: user?.email || '',
        name: user?.name || 'You',
        role: 'owner',
        joinedAt: new Date().toISOString(),
      },
    ]);
  }, [user]);

  const roleColors: Record<string, string> = {
    owner: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
    admin: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
    member: 'bg-green-500/10 text-green-600 dark:text-green-400',
    viewer: 'bg-zinc-500/10 text-zinc-600 dark:text-zinc-400',
  };

  return (
    <EnterpriseGate
      fallback={
        <div className="p-4 sm:p-6 lg:p-8 space-y-8 max-w-4xl overflow-auto h-full">
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Teams</h1>
            <p className="text-zinc-500 dark:text-zinc-400 mt-1">
              Team management is an enterprise feature.
            </p>
          </div>
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-zinc-500 dark:text-zinc-400 mb-4">
                  Upgrade to Enterprise to invite team members and collaborate.
                </p>
                <Button>Upgrade to Enterprise</Button>
              </div>
            </CardContent>
          </Card>
        </div>
      }
    >
      <div className="p-4 sm:p-6 lg:p-8 space-y-8 max-w-4xl overflow-auto h-full">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Teams</h1>
            <p className="text-zinc-500 dark:text-zinc-400 mt-1">
              Manage your team members and their roles.
            </p>
          </div>
          <Button>Invite Member</Button>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Team Members</CardTitle>
            <CardDescription>
              {members.length} member{members.length !== 1 ? 's' : ''}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-4">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="h-16 bg-zinc-200 dark:bg-zinc-800 rounded animate-pulse" />
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                {members.map((member) => (
                  <div
                    key={member.id}
                    className="flex items-center justify-between p-4 rounded-lg border border-zinc-200 dark:border-zinc-800"
                  >
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-full bg-zinc-200 dark:bg-zinc-700 flex items-center justify-center">
                        <span className="text-sm font-medium text-zinc-600 dark:text-zinc-300">
                          {member.name?.charAt(0).toUpperCase() || '?'}
                        </span>
                      </div>
                      <div>
                        <p className="font-medium text-zinc-900 dark:text-white">{member.name}</p>
                        <p className="text-sm text-zinc-500 dark:text-zinc-400">{member.email}</p>
                      </div>
                    </div>
                    <Badge className={roleColors[member.role]}>
                      {member.role}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </EnterpriseGate>
  );
}
