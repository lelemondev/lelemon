'use client';

import { useState, useCallback, memo, useMemo, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth-context';
import { useEE } from '../lib/ee-context';

export type OrganizationRole = 'owner' | 'admin' | 'member' | 'viewer';

export interface Organization {
  id: string;
  name: string;
  slug: string;
  role: OrganizationRole;
  isPersonal?: boolean;
}

interface OrganizationContextType {
  organizations: Organization[];
  currentOrganization: Organization | null;
  isLoading: boolean;
  switchOrganization: (orgId: string) => void;
}

// Role display labels and colors
const roleConfig: Record<OrganizationRole, { label: string; className: string }> = {
  owner: {
    label: 'Owner',
    className: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
  },
  admin: {
    label: 'Admin',
    className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
  },
  member: {
    label: 'Member',
    className: 'bg-green-500/10 text-green-600 dark:text-green-400',
  },
  viewer: {
    label: 'Viewer',
    className: 'bg-zinc-500/10 text-zinc-600 dark:text-zinc-400',
  },
};

interface OrganizationSwitcherProps {
  collapsed?: boolean;
}

/**
 * Organization switcher component for EE.
 * Shows current organization with role and allows switching between organizations.
 */
export const OrganizationSwitcher = memo(function OrganizationSwitcher({
  collapsed = false,
}: OrganizationSwitcherProps) {
  const { isEnterprise, hasFeature, isLoading: eeLoading } = useEE();
  const { user, token } = useAuth();
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [currentOrgId, setCurrentOrgId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isOpen, setIsOpen] = useState(false);

  // Fetch organizations from API
  useEffect(() => {
    if (!isEnterprise || !hasFeature('organizations') || !token) {
      setIsLoading(false);
      return;
    }

    const fetchOrganizations = async () => {
      try {
        const apiUrl = process.env.NEXT_PUBLIC_API_URL || '';
        const response = await fetch(`${apiUrl}/api/v1/organizations`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });

        if (response.ok) {
          const data = await response.json();
          setOrganizations(data.organizations || []);

          // Load saved org from localStorage or use first org
          const savedOrgId = localStorage.getItem('lelemon_current_org');
          const validOrg = data.organizations?.find((org: Organization) => org.id === savedOrgId);
          setCurrentOrgId(validOrg?.id || data.organizations?.[0]?.id || null);
        }
      } catch (error) {
        console.debug('Failed to fetch organizations:', error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchOrganizations();
  }, [isEnterprise, hasFeature, token]);

  // Get current organization
  const currentOrganization = useMemo(() => {
    return organizations.find((org) => org.id === currentOrgId) || null;
  }, [organizations, currentOrgId]);

  // Switch organization
  const switchOrganization = useCallback((orgId: string) => {
    setCurrentOrgId(orgId);
    localStorage.setItem('lelemon_current_org', orgId);
    setIsOpen(false);
    // Trigger page refresh or context update
    window.location.reload();
  }, []);

  // Don't render in community edition
  if (eeLoading || !isEnterprise || !hasFeature('organizations')) {
    return null;
  }

  // Loading state
  if (isLoading) {
    return (
      <div
        data-testid="organization-switcher"
        className={cn('p-3 border-b border-zinc-200 dark:border-zinc-800', collapsed && 'p-2')}
      >
        <div className="animate-pulse flex items-center gap-3">
          <div className="w-8 h-8 rounded bg-zinc-200 dark:bg-zinc-800" />
          {!collapsed && <div className="h-4 w-24 rounded bg-zinc-200 dark:bg-zinc-800" />}
        </div>
      </div>
    );
  }

  // No organizations yet
  if (organizations.length === 0) {
    return null;
  }

  // Collapsed view - just show avatar
  if (collapsed) {
    return (
      <div
        data-testid="organization-switcher"
        className="p-2 border-b border-zinc-200 dark:border-zinc-800"
      >
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="w-full flex justify-center"
          title={currentOrganization?.name || 'Switch organization'}
        >
          <div className="w-8 h-8 rounded bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center">
            <span className="text-xs font-bold text-white">
              {currentOrganization?.name?.charAt(0).toUpperCase() || 'O'}
            </span>
          </div>
        </button>
      </div>
    );
  }

  return (
    <div
      data-testid="organization-switcher"
      className="p-3 border-b border-zinc-200 dark:border-zinc-800 relative"
    >
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center gap-3 p-2 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
        role="combobox"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
      >
        {/* Organization avatar */}
        <div className="w-8 h-8 rounded bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center flex-shrink-0">
          <span className="text-xs font-bold text-white">
            {currentOrganization?.name?.charAt(0).toUpperCase() || 'O'}
          </span>
        </div>

        {/* Org name and role */}
        <div className="flex-1 min-w-0 text-left">
          <p className="text-sm font-medium text-zinc-900 dark:text-white truncate">
            {currentOrganization?.name || 'Select Organization'}
          </p>
          <div className="flex items-center gap-2">
            {currentOrganization?.isPersonal ? (
              <span className="text-xs text-zinc-500 dark:text-zinc-400">Personal</span>
            ) : currentOrganization?.role ? (
              <span
                data-testid="user-role"
                className={cn(
                  'text-xs px-1.5 py-0.5 rounded',
                  roleConfig[currentOrganization.role].className
                )}
              >
                {roleConfig[currentOrganization.role].label}
              </span>
            ) : null}
          </div>
        </div>

        {/* Chevron */}
        <svg
          className={cn(
            'w-4 h-4 text-zinc-400 transition-transform',
            isOpen && 'rotate-180'
          )}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown */}
      {isOpen && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Dropdown menu */}
          <div
            className="absolute left-3 right-3 top-full mt-1 z-50 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-lg py-1 max-h-64 overflow-y-auto"
            role="listbox"
          >
            {organizations.map((org) => (
              <button
                key={org.id}
                onClick={() => switchOrganization(org.id)}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors',
                  org.id === currentOrgId && 'bg-amber-500/10'
                )}
                role="option"
                aria-selected={org.id === currentOrgId}
              >
                {/* Org avatar */}
                <div className="w-6 h-6 rounded bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center flex-shrink-0">
                  <span className="text-[10px] font-bold text-white">
                    {org.name.charAt(0).toUpperCase()}
                  </span>
                </div>

                {/* Org info */}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-zinc-900 dark:text-white truncate">
                    {org.name}
                  </p>
                </div>

                {/* Role badge */}
                {org.isPersonal ? (
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">Personal</span>
                ) : (
                  <span
                    className={cn(
                      'text-xs px-1.5 py-0.5 rounded',
                      roleConfig[org.role].className
                    )}
                  >
                    {roleConfig[org.role].label}
                  </span>
                )}

                {/* Check mark for current */}
                {org.id === currentOrgId && (
                  <svg
                    className="w-4 h-4 text-amber-500"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                )}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
});
