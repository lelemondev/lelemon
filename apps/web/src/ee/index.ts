// Enterprise Edition Web Components
// This module exports all EE-specific components and utilities

export { EEProvider, useEE } from './lib/ee-context';
export { FeatureGate, EnterpriseGate } from './components/feature-gate';
export { EENavigation, EditionBadge } from './components/navigation';
export { OrganizationSwitcher } from './components/organization-switcher';
export type { Organization, OrganizationRole } from './components/organization-switcher';
