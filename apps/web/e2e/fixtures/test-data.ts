/**
 * Test data fixtures for E2E tests.
 */

export const testUsers = {
  // Default test user (created in setup)
  default: {
    email: 'test@lelemon.dev',
    password: 'TestPassword123!',
    name: 'Test User',
  },

  // Secondary user for multi-user tests
  secondary: {
    email: 'test2@lelemon.dev',
    password: 'TestPassword123!',
    name: 'Test User 2',
  },

  // Admin user for EE tests
  admin: {
    email: 'admin@lelemon.dev',
    password: 'AdminPassword123!',
    name: 'Admin User',
  },
};

export const testOrganizations = {
  default: {
    name: 'Test Organization',
    slug: 'test-org',
  },
  secondary: {
    name: 'Another Organization',
    slug: 'another-org',
  },
};

export const testProjects = {
  default: {
    name: 'Test Project',
  },
  secondary: {
    name: 'Another Project',
  },
};

/**
 * Generate unique test data to avoid collisions.
 */
export function uniqueEmail(prefix = 'test'): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return `${prefix}-${timestamp}-${random}@lelemon.dev`;
}

export function uniqueSlug(prefix = 'org'): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return `${prefix}-${timestamp}-${random}`;
}

export function uniqueName(prefix = 'Test'): string {
  const timestamp = Date.now();
  return `${prefix} ${timestamp}`;
}
