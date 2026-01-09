'use client';

import { ReactNode, memo } from 'react';
import { useEE } from '../lib/ee-context';

interface FeatureGateProps {
  feature: string;
  children: ReactNode;
  fallback?: ReactNode;
}

/**
 * Conditionally renders children based on whether a feature is enabled.
 * Memoized to prevent re-renders when parent re-renders with same props.
 *
 * Usage:
 * <FeatureGate feature="organizations">
 *   <OrganizationSwitcher />
 * </FeatureGate>
 */
export const FeatureGate = memo(function FeatureGate({
  feature,
  children,
  fallback = null
}: FeatureGateProps) {
  const { hasFeature, isLoading } = useEE();

  // Don't render anything while loading to avoid flash
  if (isLoading) {
    return null;
  }

  if (!hasFeature(feature)) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
});

interface EnterpriseGateProps {
  children: ReactNode;
  fallback?: ReactNode;
}

/**
 * Conditionally renders children only in enterprise edition.
 * Memoized to prevent unnecessary re-renders.
 */
export const EnterpriseGate = memo(function EnterpriseGate({
  children,
  fallback = null
}: EnterpriseGateProps) {
  const { isEnterprise, isLoading } = useEE();

  if (isLoading) {
    return null;
  }

  if (!isEnterprise) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
});
