'use client';

import { useAuth } from '@/lib/auth-context';
import { cn } from '@/lib/utils';

interface UserInfoProps {
  collapsed?: boolean;
}

export function UserInfo({ collapsed = false }: UserInfoProps) {
  const { user } = useAuth();

  if (!user) {
    return null;
  }

  // Generate initials from name
  const initials = user.name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  if (collapsed) {
    return (
      <div
        data-testid="user-info"
        className="flex justify-center py-2"
        title={`${user.name}\n${user.email}`}
      >
        <div className="w-8 h-8 rounded-full bg-amber-500/10 flex items-center justify-center">
          <span className="text-xs font-medium text-amber-600 dark:text-amber-400">
            {initials}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div data-testid="user-info" className="px-3 py-2">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-full bg-amber-500/10 flex items-center justify-center flex-shrink-0">
          <span className="text-xs font-medium text-amber-600 dark:text-amber-400">
            {initials}
          </span>
        </div>
        <div className={cn("min-w-0 flex-1")}>
          <p className="text-sm font-medium text-zinc-900 dark:text-white truncate">
            {user.name}
          </p>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
            {user.email}
          </p>
        </div>
      </div>
    </div>
  );
}
