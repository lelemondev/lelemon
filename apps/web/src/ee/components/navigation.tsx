'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { memo, useMemo } from 'react';
import { cn } from '@/lib/utils';
import { useEE } from '../lib/ee-context';

interface EENavigationProps {
  collapsed?: boolean;
  onNavigate?: () => void;
}

const eeNavigation = [
  {
    name: 'Teams',
    href: '/dashboard/teams',
    feature: 'organizations',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
      </svg>
    )
  },
  {
    name: 'Billing',
    href: '/dashboard/billing',
    feature: 'billing',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" />
      </svg>
    )
  },
];

/**
 * Enterprise navigation items for the sidebar.
 * Only renders items for features that are enabled.
 * Memoized to prevent unnecessary re-renders.
 */
export const EENavigation = memo(function EENavigation({
  collapsed = false,
  onNavigate
}: EENavigationProps) {
  const pathname = usePathname();
  const { hasFeature, isLoading, features } = useEE();

  // Memoize filtered items to avoid recalculating on every render
  const enabledItems = useMemo(() =>
    eeNavigation.filter(item => hasFeature(item.feature)),
    [features, hasFeature]
  );

  // Don't render anything while loading
  if (isLoading) {
    return null;
  }

  if (enabledItems.length === 0) {
    return null;
  }

  return (
    <>
      {/* Separator */}
      <div className={cn(
        "border-t border-zinc-200 dark:border-zinc-800 my-2",
        collapsed ? "mx-2" : "mx-3"
      )} />

      {/* EE Navigation Items */}
      {enabledItems.map((item) => {
        const isActive = pathname === item.href ||
          (item.href !== '/dashboard' && pathname.startsWith(item.href));

        return (
          <Link
            key={item.name}
            href={item.href}
            onClick={onNavigate}
            title={collapsed ? item.name : undefined}
            className={cn(
              'flex items-center rounded-lg text-sm font-medium transition-all',
              collapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2.5',
              isActive
                ? 'bg-amber-500/10 text-zinc-900 dark:text-white'
                : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-900 dark:hover:text-white'
            )}
          >
            <span className={cn(
              'transition-colors flex-shrink-0',
              isActive ? 'text-amber-500' : 'text-zinc-400 dark:text-zinc-500'
            )}>
              {item.icon}
            </span>
            {!collapsed && item.name}
          </Link>
        );
      })}
    </>
  );
});

/**
 * Enterprise badge/label showing the current edition.
 * Memoized to prevent unnecessary re-renders.
 */
export const EditionBadge = memo(function EditionBadge() {
  const { edition, isLoading } = useEE();

  if (isLoading || edition === 'community') {
    return null;
  }

  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400">
      Enterprise
    </span>
  );
});
