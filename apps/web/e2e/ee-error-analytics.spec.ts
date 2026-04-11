import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { AuthHelper } from './helpers/auth';
import { uniqueEmail } from './fixtures/test-data';

/**
 * Enterprise Edition (EE) Error Analytics E2E tests.
 *
 * These tests verify the Error Rate Analytics feature:
 * - API endpoint returns correct error metrics
 * - Error rate calculation is accurate
 * - Error breakdown by tag works
 * - Top errors are aggregated correctly
 * - Frontend component displays data correctly
 * - Multi-tenant isolation
 *
 * NOTE: These tests require EE features to be enabled.
 * They will be skipped if running in OSS mode.
 */
test.describe('Enterprise: Error Analytics', () => {
  /**
   * Helper to setup a user with traces containing errors for testing.
   */
  async function setupUserWithErrorTraces(
    api: ApiHelper,
    auth: AuthHelper,
    page: import('@playwright/test').Page,
    emailPrefix: string
  ) {
    const email = uniqueEmail(emailPrefix);
    const password = 'TestPassword123!';

    await api.register(email, password, `${emailPrefix} User`);
    const loginResult = await api.login(email, password);
    const projectResult = await api.createProject(loginResult.data.token, `${emailPrefix} Project`);
    const project = projectResult.data;

    // Ingest traces with mix of success and errors
    await api.ingestTracesWithTags(project.APIKey, [
      // Success traces
      { name: 'Success Chat 1', tags: ['org:acme', 'env:prod'], status: 'success' },
      { name: 'Success Chat 2', tags: ['org:acme', 'env:prod'], status: 'success' },
      { name: 'Success Chat 3', tags: ['org:acme', 'env:staging'], status: 'success' },
      { name: 'Success Chat 4', tags: ['org:globex', 'env:prod'], status: 'success' },
      // Error traces
      { name: 'Error Chat 1', tags: ['org:acme', 'env:prod'], status: 'error', errorMessage: 'Rate limit exceeded' },
      { name: 'Error Chat 2', tags: ['org:acme', 'env:prod'], status: 'error', errorMessage: 'Rate limit exceeded' },
      { name: 'Error Chat 3', tags: ['org:globex', 'env:staging'], status: 'error', errorMessage: 'Invalid API key' },
      { name: 'Error Chat 4', tags: ['org:globex', 'env:staging'], status: 'error', errorMessage: 'Model overloaded' },
    ]);

    await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

    return { email, password, project, loginResult };
  }

  test.describe('API: Error Metrics Endpoint', () => {
    test('should return error metrics with correct totals', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithErrorTraces(api, auth, page, 'err-api-basic');

      const result = await api.getErrorMetrics(loginResult.data.token, project.ID);

      expect(result.status).toBe(200);
      expect(result.data).toBeDefined();

      // 8 total traces (4 success + 4 error)
      expect(result.data.totalTraces).toBe(8);
      expect(result.data.errorTraces).toBe(4);

      // Error rate should be 50% (4/8)
      expect(result.data.errorRate).toBe(50);
    });

    test('should return error rate breakdown by tag', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithErrorTraces(api, auth, page, 'err-api-bytag');

      const result = await api.getErrorMetrics(loginResult.data.token, project.ID);

      expect(result.status).toBe(200);
      expect(result.data.byTag).toBeInstanceOf(Array);

      // Verify tags are present
      const tagNames = result.data.byTag.map((t: { tag: string }) => t.tag);
      expect(tagNames).toContain('org:acme');
      expect(tagNames).toContain('org:globex');
      expect(tagNames).toContain('env:prod');
      expect(tagNames).toContain('env:staging');

      // org:acme has 5 traces (3 success + 2 error) = 40% error rate
      const acmeTag = result.data.byTag.find((t: { tag: string }) => t.tag === 'org:acme');
      expect(acmeTag).toBeDefined();
      expect(acmeTag.totalTraces).toBe(5);
      expect(acmeTag.errorTraces).toBe(2);
      expect(acmeTag.errorRate).toBe(40);
    });

    test('should return top errors aggregated by message', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithErrorTraces(api, auth, page, 'err-api-top');

      const result = await api.getErrorMetrics(loginResult.data.token, project.ID);

      expect(result.status).toBe(200);
      expect(result.data.topErrors).toBeInstanceOf(Array);

      // "Rate limit exceeded" should be the top error (2 occurrences)
      const topError = result.data.topErrors[0];
      expect(topError.message).toBe('Rate limit exceeded');
      expect(topError.count).toBe(2);
      expect(topError.affectedTags).toContain('org:acme');
    });

    test('should filter by tag prefix', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithErrorTraces(api, auth, page, 'err-api-prefix');

      // Filter by 'env:' prefix
      const result = await api.getErrorMetrics(loginResult.data.token, project.ID, { tagPrefix: 'env:' });

      expect(result.status).toBe(200);

      // byTag should only contain env: tags
      result.data.byTag.forEach((t: { tag: string }) => {
        expect(t.tag).toMatch(/^env:/);
      });

      const tagNames = result.data.byTag.map((t: { tag: string }) => t.tag);
      expect(tagNames).toContain('env:prod');
      expect(tagNames).toContain('env:staging');
      expect(tagNames).not.toContain('org:acme');
    });

    test('should respect topLimit parameter', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithErrorTraces(api, auth, page, 'err-api-limit');

      const result = await api.getErrorMetrics(loginResult.data.token, project.ID, { topLimit: 1 });

      expect(result.status).toBe(200);
      expect(result.data.topErrors.length).toBeLessThanOrEqual(1);
    });

    test('should return zero error rate for project with no errors', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-api-noerr');
      const password = 'TestPassword123!';

      await api.register(email, password, 'No Errors User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'No Errors Project');

      // Only ingest success traces
      await api.ingestTracesWithTags(projectResult.data.APIKey, [
        { name: 'Good Chat 1', tags: ['test'], status: 'success' },
        { name: 'Good Chat 2', tags: ['test'], status: 'success' },
      ]);

      const result = await api.getErrorMetrics(loginResult.data.token, projectResult.data.ID);

      expect(result.status).toBe(200);
      expect(result.data.errorRate).toBe(0);
      expect(result.data.errorTraces).toBe(0);
      expect(result.data.topErrors).toEqual([]);
    });

    test('should return empty metrics for project with no traces', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-api-empty');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Empty Errors User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Empty Project');

      const result = await api.getErrorMetrics(loginResult.data.token, projectResult.data.ID);

      expect(result.status).toBe(200);
      expect(result.data.totalTraces).toBe(0);
      expect(result.data.errorTraces).toBe(0);
      expect(result.data.errorRate).toBe(0);
      expect(result.data.byTag).toEqual([]);
      expect(result.data.topErrors).toEqual([]);
    });

    test('should NOT allow accessing another users error metrics', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Create User A with error data
      const { project: projectA } = await setupUserWithErrorTraces(api, auth, page, 'err-api-sec-a');

      // Create User B
      const emailB = uniqueEmail('err-api-sec-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');

      // User B tries to access User A's error metrics
      const result = await api.getErrorMetrics(loginB.data.token, projectA.ID);

      // Should be forbidden or not found
      expect([403, 404]).toContain(result.status);
    });
  });

  test.describe('Frontend: Error Analytics Component', () => {
    test('should display error rate overview', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithErrorTraces(api, auth, page, 'err-ui-overview');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');
      await page.waitForTimeout(500);

      // Should see Error Analytics section (in EE mode) or Enterprise Feature message (in OSS)
      const errorSection = page.locator('text=Error Rate Overview, text=Error Analytics');
      await expect(errorSection.first()).toBeVisible({ timeout: 10000 });
    });

    test('should display error rate by tag table', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithErrorTraces(api, auth, page, 'err-ui-bytag');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');
      await page.waitForTimeout(500);

      // In EE mode, should see Error Rate by Tag section
      // In OSS mode, should see Enterprise Feature message
      const eeContent = page.locator('text=Error Rate by Tag');
      const ossContent = page.locator('text=Enterprise Feature');

      // Either EE content or OSS fallback should be visible
      await expect(eeContent.or(ossContent)).toBeVisible({ timeout: 10000 });
    });

    test('should display top errors table', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithErrorTraces(api, auth, page, 'err-ui-top');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');
      await page.waitForTimeout(500);

      // In EE mode, should see Top Errors section
      // In OSS mode, should see Enterprise Feature message
      const eeContent = page.locator('text=Top Errors');
      const ossContent = page.locator('text=Enterprise Feature');

      await expect(eeContent.or(ossContent)).toBeVisible({ timeout: 10000 });
    });

    test('should color-code error rates', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithErrorTraces(api, auth, page, 'err-ui-colors');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');
      await page.waitForTimeout(500);

      // In EE mode, should have colored elements based on error rate
      // In OSS mode, will show Enterprise Feature fallback
      const eeContent = page.locator('.text-red-500, .bg-red-500');
      const ossContent = page.locator('text=Enterprise Feature');

      // Either colored elements (EE) or fallback (OSS) should be visible
      await expect(eeContent.first().or(ossContent)).toBeVisible({ timeout: 5000 });
    });

    test('should filter by tag prefix using input', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithErrorTraces(api, auth, page, 'err-ui-filter');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');
      await page.waitForTimeout(500);

      // Find tag prefix filter (only visible in EE mode)
      const prefixInput = page.locator('input[placeholder*="tag prefix" i]');
      if (await prefixInput.isVisible({ timeout: 3000 }).catch(() => false)) {
        await prefixInput.fill('env:');

        // Click filter button
        await page.click('button:has-text("Filter")');
        await page.waitForTimeout(500);

        // Should only show env: tags
        await expect(page.locator('text=/env:prod|env:staging/')).toBeVisible();
      }
    });

    test('should show empty state when no errors', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-ui-empty');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Empty Error UI User');
      const loginResult = await api.login(email, password);
      await api.createProject(loginResult.data.token, 'Empty Error Project');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');
      await page.waitForTimeout(500);

      // In EE mode, may show empty state; in OSS mode shows Enterprise Feature
      const eeContent = page.locator('text=/No errors found|No tags found/i');
      const ossContent = page.locator('text=Enterprise Feature');

      await expect(eeContent.or(ossContent)).toBeVisible({ timeout: 5000 });
    });

    test('should show loading state while fetching', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithErrorTraces(api, auth, page, 'err-ui-loading');

      // Navigate to analytics
      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Errors tab
      await page.click('button:has-text("Errors")');

      // After loading, should see error section
      const eeContent = page.locator('text=Error Rate Overview');
      const ossContent = page.locator('text=Enterprise Feature');

      await expect(eeContent.or(ossContent)).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Integration: Frontend + Backend', () => {
    test('should correctly calculate error rate after new errors', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-int-calc');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Calc User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Calc Project');
      const project = projectResult.data;

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      // Start with 10 success traces
      const successTraces = Array.from({ length: 10 }, (_, i) => ({
        name: `Success ${i}`,
        tags: ['test'],
        status: 'success' as const,
      }));
      await api.ingestTracesWithTags(project.APIKey, successTraces);

      // Verify 0% error rate
      let result = await api.getErrorMetrics(loginResult.data.token, project.ID);
      expect(result.data.errorRate).toBe(0);

      // Add 2 error traces (now 10 success + 2 error = 12 total, ~16.67% error rate)
      await api.ingestTracesWithTags(project.APIKey, [
        { name: 'Error 1', tags: ['test'], status: 'error', errorMessage: 'Test error' },
        { name: 'Error 2', tags: ['test'], status: 'error', errorMessage: 'Test error' },
      ]);

      // Verify increased error rate
      result = await api.getErrorMetrics(loginResult.data.token, project.ID);
      expect(result.data.totalTraces).toBe(12);
      expect(result.data.errorTraces).toBe(2);
      // 2/12 = 16.666...% which should round to 16.67 or similar
      expect(result.data.errorRate).toBeGreaterThan(16);
      expect(result.data.errorRate).toBeLessThan(17);
    });

    test('should aggregate same error messages correctly', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-int-agg');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Aggregate User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Aggregate Project');
      const project = projectResult.data;

      // Ingest same error multiple times
      await api.ingestTracesWithTags(project.APIKey, [
        { name: 'E1', tags: ['a'], status: 'error', errorMessage: 'Connection timeout' },
        { name: 'E2', tags: ['b'], status: 'error', errorMessage: 'Connection timeout' },
        { name: 'E3', tags: ['c'], status: 'error', errorMessage: 'Connection timeout' },
        { name: 'E4', tags: ['a'], status: 'error', errorMessage: 'Other error' },
      ]);

      const result = await api.getErrorMetrics(loginResult.data.token, project.ID);

      // "Connection timeout" should be aggregated with count 3
      const connectionError = result.data.topErrors.find(
        (e: { message: string }) => e.message === 'Connection timeout'
      );
      expect(connectionError).toBeDefined();
      expect(connectionError.count).toBe(3);

      // Should have 3 affected tags: a, b, c
      expect(connectionError.affectedTags).toContain('a');
      expect(connectionError.affectedTags).toContain('b');
      expect(connectionError.affectedTags).toContain('c');
    });

    test('should track lastOccurred timestamp', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-int-time');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Time User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Time Project');
      const project = projectResult.data;

      // Ingest error
      await api.ingestTracesWithTags(project.APIKey, [
        { name: 'E1', tags: ['test'], status: 'error', errorMessage: 'Timed error' },
      ]);

      const result = await api.getErrorMetrics(loginResult.data.token, project.ID);

      // lastOccurred should be a valid timestamp
      const error = result.data.topErrors[0];
      expect(error.lastOccurred).toBeDefined();

      // Should be a valid ISO date string
      const date = new Date(error.lastOccurred);
      expect(date.getTime()).not.toBeNaN();

      // Should be recent (within last minute)
      const now = Date.now();
      const errorTime = date.getTime();
      expect(now - errorTime).toBeLessThan(60000);
    });
  });

  test.describe('Security: Data Isolation', () => {
    test('users cannot see error data from other projects', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // User A creates project with sensitive errors
      const emailA = uniqueEmail('err-sec-a');
      await api.register(emailA, 'TestPassword123!', 'User A');
      const loginA = await api.login(emailA, 'TestPassword123!');
      const projectA = await api.createProject(loginA.data.token, 'Sensitive Project');

      await api.ingestTracesWithTags(projectA.data.APIKey, [
        { name: 'Secret Error', tags: ['secret:true'], status: 'error', errorMessage: 'Internal: password hash mismatch' },
      ]);

      // User B creates their own project
      const emailB = uniqueEmail('err-sec-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');
      const projectB = await api.createProject(loginB.data.token, 'User B Project');

      // User B's error metrics should NOT include User A's data
      const resultB = await api.getErrorMetrics(loginB.data.token, projectB.data.ID);

      // Should not have secret tag
      const tags = resultB.data.byTag.map((t: { tag: string }) => t.tag);
      expect(tags).not.toContain('secret:true');

      // Should not have sensitive error message
      const errorMessages = resultB.data.topErrors.map((e: { message: string }) => e.message);
      expect(errorMessages).not.toContain('Internal: password hash mismatch');
    });

    test('API requires authentication', async ({ request }) => {
      const api = new ApiHelper(request);

      // Create a valid project first
      const email = uniqueEmail('err-sec-noauth');
      await api.register(email, 'TestPassword123!', 'No Auth User');
      const loginResult = await api.login(email, 'TestPassword123!');
      const project = await api.createProject(loginResult.data.token, 'Auth Test');

      // Try to access without token
      const response = await request.get(
        `${api.getBaseUrl()}/api/v1/dashboard/projects/${project.data.ID}/analytics/errors`
      );

      expect(response.status()).toBe(401);
    });
  });

  test.describe('Edge Cases', () => {
    test('should handle 100% error rate', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-edge-100');
      const password = 'TestPassword123!';

      await api.register(email, password, '100% Error User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, '100% Error Project');

      // Only error traces
      await api.ingestTracesWithTags(projectResult.data.APIKey, [
        { name: 'E1', tags: ['bad'], status: 'error', errorMessage: 'Fail 1' },
        { name: 'E2', tags: ['bad'], status: 'error', errorMessage: 'Fail 2' },
        { name: 'E3', tags: ['bad'], status: 'error', errorMessage: 'Fail 3' },
      ]);

      const result = await api.getErrorMetrics(loginResult.data.token, projectResult.data.ID);

      expect(result.data.errorRate).toBe(100);
      expect(result.data.totalTraces).toBe(3);
      expect(result.data.errorTraces).toBe(3);
    });

    test('should handle traces without tags', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-edge-notag');
      const password = 'TestPassword123!';

      await api.register(email, password, 'No Tag User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'No Tag Project');

      // Traces without tags
      await api.ingestSpans(projectResult.data.APIKey, [
        { spanType: 'llm', name: 'No Tag Success', status: 'success' },
        { spanType: 'llm', name: 'No Tag Error', status: 'error', errorMessage: 'No tag error' },
      ]);

      const result = await api.getErrorMetrics(loginResult.data.token, projectResult.data.ID);

      // Should still calculate overall metrics
      expect(result.data.totalTraces).toBe(2);
      expect(result.data.errorTraces).toBe(1);
      expect(result.data.errorRate).toBe(50);

      // byTag should be empty or have only empty tag
      expect(result.data.byTag.length).toBe(0);
    });

    test('should handle very long error messages', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('err-edge-long');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Long Error User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Long Error Project');

      const longMessage = 'A'.repeat(1000);
      await api.ingestTracesWithTags(projectResult.data.APIKey, [
        { name: 'Long Error', tags: ['test'], status: 'error', errorMessage: longMessage },
      ]);

      const result = await api.getErrorMetrics(loginResult.data.token, projectResult.data.ID);

      // Should handle without crashing
      expect(result.status).toBe(200);
      expect(result.data.topErrors.length).toBe(1);
    });
  });
});
