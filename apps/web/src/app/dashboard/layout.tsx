'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { ThemeToggle } from '@/components/theme-toggle';

const navigation = [
  { name: 'Overview', href: '/dashboard' },
  { name: 'Traces', href: '/dashboard/traces' },
  { name: 'Analytics', href: '/dashboard/analytics' },
  { name: 'Settings', href: '/dashboard/settings' },
];

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950">
      {/* Header */}
      <header className="sticky top-0 z-50 w-full border-b bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:bg-zinc-950/95 dark:border-zinc-800">
        <div className="flex h-14 items-center px-6">
          <Link href="/dashboard" className="flex items-center space-x-2">
            <span className="text-xl font-bold text-lime-500">🍋</span>
            <span className="font-bold">Lelemon</span>
          </Link>
          <nav className="ml-8 flex items-center space-x-6 text-sm font-medium">
            {navigation.map((item) => (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  'transition-colors hover:text-foreground/80',
                  pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(item.href))
                    ? 'text-foreground'
                    : 'text-foreground/60'
                )}
              >
                {item.name}
              </Link>
            ))}
          </nav>
          <div className="ml-auto flex items-center space-x-4">
            <span className="text-xs font-mono text-zinc-500 dark:text-zinc-400 bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">
              Demo Project
            </span>
            <ThemeToggle />
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="p-6">
        {children}
      </main>
    </div>
  );
}
