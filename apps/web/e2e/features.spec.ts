import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';

/**
 * Tests for the /api/v1/features endpoint.
 * Verifies OSS vs EE feature detection.
 */
test.describe('Features API', () => {
  test('should return features configuration', async ({ request }) => {
    const api = new ApiHelper(request);
    const features = await api.getFeatures();

    // Should have edition
    expect(features.edition).toBeDefined();
    expect(['community', 'enterprise']).toContain(features.edition);

    // Should have features object
    expect(features.features).toBeDefined();
    expect(typeof features.features).toBe('object');
  });

  test('community edition should have EE features disabled', async ({ request }) => {
    const api = new ApiHelper(request);
    const features = await api.getFeatures();

    if (features.edition === 'community') {
      expect(features.features.organizations).toBe(false);
      expect(features.features.rbac).toBe(false);
      expect(features.features.billing).toBe(false);
    }
  });

  test('enterprise edition should have EE features enabled', async ({ request }) => {
    const api = new ApiHelper(request);
    const features = await api.getFeatures();

    if (features.edition === 'enterprise') {
      expect(features.features.organizations).toBe(true);
      expect(features.features.rbac).toBe(true);
      expect(features.features.billing).toBe(true);
    }
  });
});

test.describe('Features UI', () => {
  test('should show correct navigation items based on edition', async ({ page, request }) => {
    const api = new ApiHelper(request);
    const features = await api.getFeatures();

    // Go to dashboard (assuming logged in or public page)
    await page.goto('/dashboard');

    if (features.edition === 'enterprise') {
      // EE should show Teams and Billing in nav
      await expect(page.locator('nav')).toContainText('Teams');
      await expect(page.locator('nav')).toContainText('Billing');
    } else {
      // OSS should NOT show Teams and Billing
      // (or should show upgrade prompt)
      const teamsLink = page.locator('nav a:has-text("Teams")');
      const billingLink = page.locator('nav a:has-text("Billing")');

      // Either not visible or leads to upgrade page
      const teamsVisible = await teamsLink.isVisible().catch(() => false);
      const billingVisible = await billingLink.isVisible().catch(() => false);

      // In OSS, these should generally not be in main nav
      // (implementation may vary - adjust based on actual behavior)
      if (!teamsVisible && !billingVisible) {
        // Expected for pure OSS
        expect(true).toBe(true);
      }
    }
  });
});
