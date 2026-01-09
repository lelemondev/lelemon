'use client';

import { useState, useEffect, createContext, useContext, useMemo, Fragment } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import dynamic from 'next/dynamic';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth-context';
import { ThemeToggle } from '@/components/theme-toggle';
import { ProjectProvider } from '@/lib/project-context';
import { ProjectSelector } from '@/components/project-selector';
import { LemonIcon } from '@/components/lemon-icon';
import { UserInfo } from '@/components/user-info';

// Dynamic imports for EE components - fail silently if not available
const EEProvider = dynamic(
  () => import('@/ee/lib/ee-context').then(m => m.EEProvider).catch(() => Fragment),
  { ssr: false }
);
const EENavigation = dynamic(
  () => import('@/ee/components/navigation').then(m => m.EENavigation).catch(() => () => null),
  { ssr: false }
);
const OrganizationSwitcher = dynamic(
  () => import('@/ee/components/organization-switcher').then(m => m.OrganizationSwitcher).catch(() => () => null),
  { ssr: false }
);

// Sidebar context for children to know if collapsed
const SidebarContext = createContext({ collapsed: false, setCollapsed: (_: boolean) => {} });
export const useSidebar = () => useContext(SidebarContext);

const navigation = [
  {
    name: 'Overview',
    href: '/dashboard',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
      </svg>
    )
  },
  {
    name: 'Traces',
    href: '/dashboard/traces',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z" />
      </svg>
    )
  },
  {
    name: 'Sessions',
    href: '/dashboard/sessions',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
      </svg>
    )
  },
  {
    name: 'Analytics',
    href: '/dashboard/analytics',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
      </svg>
    )
  },
  {
    name: 'Config',
    href: '/dashboard/config',
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" />
        <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    )
  },
];

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { logout, isAuthenticated, isLoading } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // User's manual preference (persisted)
  const [userCollapsed, setUserCollapsed] = useState(false);
  // Temporary auto-collapse for trace detail view
  const [autoCollapsed, setAutoCollapsed] = useState(false);

  // Detect if we're on a trace detail page
  const isTraceDetailPage = pathname.startsWith('/dashboard/traces/') && pathname !== '/dashboard/traces';

  // Load user's sidebar preference from localStorage
  useEffect(() => {
    const stored = localStorage.getItem('sidebar-collapsed');
    if (stored !== null) {
      setUserCollapsed(stored === 'true');
    }
  }, []);

  // Auto-collapse when entering trace detail, restore when leaving
  useEffect(() => {
    if (isTraceDetailPage && !userCollapsed) {
      setAutoCollapsed(true);
    } else {
      setAutoCollapsed(false);
    }
  }, [isTraceDetailPage, userCollapsed]);

  // Effective collapsed state: user preference OR auto-collapse
  const sidebarCollapsed = userCollapsed || autoCollapsed;

  // Toggle user preference (persisted)
  const toggleSidebarCollapsed = () => {
    const newValue = !userCollapsed;
    setUserCollapsed(newValue);
    localStorage.setItem('sidebar-collapsed', String(newValue));
    // If manually expanding while on detail page, cancel auto-collapse
    if (!newValue) {
      setAutoCollapsed(false);
    }
  };

  // Context value for children
  const sidebarContextValue = useMemo(() => ({
    collapsed: sidebarCollapsed,
    setCollapsed: (value: boolean) => {
      setUserCollapsed(value);
      localStorage.setItem('sidebar-collapsed', String(value));
    }
  }), [sidebarCollapsed]);

  // Redirect to login if not authenticated
  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, isLoading, router]);

  // Show loading state while checking auth
  if (isLoading) {
    return (
      <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-amber-500" />
      </div>
    );
  }

  // Don't render dashboard if not authenticated
  if (!isAuthenticated) {
    return null;
  }

  // Sidebar width constants
  const sidebarWidth = sidebarCollapsed ? 'w-16' : 'w-60';
  const mainMargin = sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-60';

  return (
    <EEProvider>
      <ProjectProvider>
        <SidebarContext.Provider value={sidebarContextValue}>
          <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950">
          {/* Mobile header */}
          <div className="lg:hidden fixed top-0 left-0 right-0 z-40 h-14 bg-white dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between px-4">
            <button
              onClick={() => setSidebarOpen(true)}
              className="p-2 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
            >
              <svg className="w-5 h-5 text-zinc-600 dark:text-zinc-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
              </svg>
            </button>
            <Link href="/dashboard" className="flex items-center gap-2">
              <LemonIcon className="w-6 h-6" />
              <span className="font-semibold text-zinc-900 dark:text-white">Lelemon</span>
            </Link>
            <div className="w-9" />
          </div>

          {/* Mobile sidebar overlay */}
          {sidebarOpen && (
            <div
              className="lg:hidden fixed inset-0 z-40 bg-black/50"
              onClick={() => setSidebarOpen(false)}
            />
          )}

          {/* Sidebar */}
          <aside className={cn(
            "fixed left-0 top-0 bottom-0 bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col z-50 transition-all duration-200",
            sidebarWidth,
            sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
          )}>
            {/* Logo */}
            <div className={cn(
              "h-14 border-b border-zinc-200 dark:border-zinc-800 flex items-center",
              sidebarCollapsed ? "justify-center px-2" : "justify-between px-4"
            )}>
              <Link href="/dashboard" className="flex items-center gap-2 group" onClick={() => setSidebarOpen(false)}>
                <LemonIcon className={cn("transition-transform group-hover:rotate-12", sidebarCollapsed ? "w-7 h-7" : "w-7 h-7")} />
                {!sidebarCollapsed && (
                  <span className="font-bold text-lg text-zinc-900 dark:text-white">Lelemon</span>
                )}
              </Link>
              <button
                onClick={() => setSidebarOpen(false)}
                className="lg:hidden p-1.5 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
              >
                <svg className="w-4 h-4 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* Organization Switcher - EE only */}
            <OrganizationSwitcher collapsed={sidebarCollapsed} />

            {/* Project Selector - only when expanded */}
            {!sidebarCollapsed && <ProjectSelector />}

            {/* Navigation */}
            <nav className={cn("flex-1 py-3 space-y-0.5", sidebarCollapsed ? "px-2" : "px-3")}>
              {navigation.map((item) => {
                const isActive = pathname === item.href ||
                  (item.href !== '/dashboard' && pathname.startsWith(item.href));

                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    onClick={() => setSidebarOpen(false)}
                    title={sidebarCollapsed ? item.name : undefined}
                    className={cn(
                      'flex items-center rounded-lg text-sm font-medium transition-all',
                      sidebarCollapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2.5',
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
                    {!sidebarCollapsed && item.name}
                  </Link>
                );
              })}

              {/* Enterprise navigation items */}
              <EENavigation
                collapsed={sidebarCollapsed}
                onNavigate={() => setSidebarOpen(false)}
              />
            </nav>

            {/* Bottom section */}
            <div className={cn("border-t border-zinc-200 dark:border-zinc-800", sidebarCollapsed ? "p-2" : "p-3")}>
              {/* User info */}
              <UserInfo collapsed={sidebarCollapsed} />

              {/* Theme toggle */}
              {!sidebarCollapsed ? (
                <div className="flex items-center justify-between px-3 py-2 mb-1">
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">Theme</span>
                  <ThemeToggle />
                </div>
              ) : (
                <div className="flex justify-center py-2 mb-1">
                  <ThemeToggle />
                </div>
              )}

              {/* Account */}
              <Link
                href="/dashboard/account"
                onClick={() => setSidebarOpen(false)}
                title={sidebarCollapsed ? 'Account' : undefined}
                className={cn(
                  'flex items-center rounded-lg text-sm font-medium transition-all',
                  sidebarCollapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2.5',
                  pathname === '/dashboard/account'
                    ? 'bg-amber-500/10 text-zinc-900 dark:text-white'
                    : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-900 dark:hover:text-white'
                )}
              >
                <svg className={cn('w-5 h-5 flex-shrink-0', pathname === '/dashboard/account' ? 'text-amber-500' : 'text-zinc-400 dark:text-zinc-500')} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M17.982 18.725A7.488 7.488 0 0012 15.75a7.488 7.488 0 00-5.982 2.975m11.963 0a9 9 0 10-11.963 0m11.963 0A8.966 8.966 0 0112 21a8.966 8.966 0 01-5.982-2.275M15 9.75a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                {!sidebarCollapsed && 'Account'}
              </Link>

              {/* Logout */}
              <button
                type="button"
                onClick={logout}
                title={sidebarCollapsed ? 'Logout' : undefined}
                className={cn(
                  'w-full flex items-center rounded-lg text-sm font-medium text-zinc-600 dark:text-zinc-400 hover:bg-red-50 dark:hover:bg-red-950 hover:text-red-600 dark:hover:text-red-400 transition-all',
                  sidebarCollapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2.5'
                )}
              >
                <svg className="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" />
                </svg>
                {!sidebarCollapsed && 'Logout'}
              </button>

            </div>

            {/* Collapse toggle - floating button on sidebar edge (desktop only) */}
            <button
              type="button"
              onClick={toggleSidebarCollapsed}
              className="hidden lg:flex absolute -right-3 top-20 z-10 w-6 h-6 items-center justify-center rounded-full bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 shadow-sm hover:bg-zinc-50 dark:hover:bg-zinc-700 hover:scale-110 transition-all"
              title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            >
              <svg
                className={cn("w-3.5 h-3.5 text-zinc-500 dark:text-zinc-400 transition-transform", sidebarCollapsed && "rotate-180")}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
              </svg>
            </button>
          </aside>

          {/* Main content - no padding, full height */}
          <main className={cn("h-screen pt-14 lg:pt-0 flex flex-col transition-all duration-200", mainMargin)}>
            {children}
          </main>
        </div>
      </SidebarContext.Provider>
    </ProjectProvider>
    </EEProvider>
  );
}
