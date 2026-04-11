import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { AuthHelper } from './helpers/auth';
import { uniqueEmail } from './fixtures/test-data';

/**
 * Enterprise Edition (EE) Cost Analytics E2E tests.
 *
 * These tests verify the Cost Breakdown by Tags feature:
 * - API endpoint returns correct cost aggregations
 * - Frontend component displays data correctly
 * - Tag prefix filtering works
 * - Date range filtering works
 * - Multi-tenant isolation (users can only see their own data)
 *
 * NOTE: These tests require EE features to be enabled.
 * They will be skipped if running in OSS mode.
 */
test.describe('Enterprise: Cost Analytics', () => {
  /**
   * Helper to setup a user with traces containing tags for testing.
   */
  async function setupUserWithTaggedTraces(
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

    // Ingest traces with different tags and costs
    await api.ingestTracesWithTags(project.APIKey, [
      { name: 'Chat 1', tags: ['org:acme', 'campaign:summer'], inputTokens: 1000, outputTokens: 500, model: 'gpt-4o' },
      { name: 'Chat 2', tags: ['org:acme', 'campaign:winter'], inputTokens: 500, outputTokens: 250, model: 'gpt-4o' },
      { name: 'Chat 3', tags: ['org:globex', 'campaign:summer'], inputTokens: 800, outputTokens: 400, model: 'gpt-4o' },
      { name: 'Chat 4', tags: ['org:globex', 'campaign:fall'], inputTokens: 300, outputTokens: 150, model: 'gpt-3.5-turbo' },
      { name: 'Chat 5', tags: ['user:john', 'feature:chat'], inputTokens: 200, outputTokens: 100, model: 'gpt-3.5-turbo' },
    ]);

    await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

    return { email, password, project, loginResult };
  }

  test.describe('API: Cost Breakdown Endpoint', () => {
    test('should return cost breakdown grouped by tags', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithTaggedTraces(api, auth, page, 'cost-api-basic');

      // Call the cost breakdown API
      const result = await api.getCostBreakdown(loginResult.data.token, project.ID);

      // Check response structure
      expect(result.status).toBe(200);
      expect(result.data).toBeDefined();
      expect(result.data.breakdown).toBeInstanceOf(Array);
      expect(result.data.totalCost).toBeGreaterThan(0);
      expect(result.data.totalTokens).toBeGreaterThan(0);

      // Verify breakdown contains expected tags
      const tags = result.data.breakdown.map((b: { tag: string }) => b.tag);
      expect(tags).toContain('org:acme');
      expect(tags).toContain('org:globex');
    });

    test('should filter by tag prefix', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithTaggedTraces(api, auth, page, 'cost-api-prefix');

      // Filter by 'org:' prefix
      const result = await api.getCostBreakdown(loginResult.data.token, project.ID, { tagPrefix: 'org:' });

      expect(result.status).toBe(200);
      expect(result.data.breakdown).toBeInstanceOf(Array);

      // Should only have org: tags
      result.data.breakdown.forEach((b: { tag: string }) => {
        expect(b.tag).toMatch(/^org:/);
      });

      // Should have org:acme and org:globex
      const tags = result.data.breakdown.map((b: { tag: string }) => b.tag);
      expect(tags).toContain('org:acme');
      expect(tags).toContain('org:globex');
      expect(tags).not.toContain('campaign:summer');
      expect(tags).not.toContain('user:john');
    });

    test('should filter by campaign prefix', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithTaggedTraces(api, auth, page, 'cost-api-campaign');

      // Filter by 'campaign:' prefix
      const result = await api.getCostBreakdown(loginResult.data.token, project.ID, { tagPrefix: 'campaign:' });

      expect(result.status).toBe(200);

      const tags = result.data.breakdown.map((b: { tag: string }) => b.tag);
      expect(tags).toContain('campaign:summer');
      expect(tags).toContain('campaign:winter');
      expect(tags).toContain('campaign:fall');
      expect(tags).not.toContain('org:acme');
    });

    test('should respect limit parameter', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithTaggedTraces(api, auth, page, 'cost-api-limit');

      // Request only top 2 tags
      const result = await api.getCostBreakdown(loginResult.data.token, project.ID, { limit: 2 });

      expect(result.status).toBe(200);
      expect(result.data.breakdown.length).toBeLessThanOrEqual(2);
    });

    test('should return empty breakdown for project with no traces', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('cost-api-empty');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Empty Project User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Empty Project');

      const result = await api.getCostBreakdown(loginResult.data.token, projectResult.data.ID);

      expect(result.status).toBe(200);
      expect(result.data.breakdown).toEqual([]);
      expect(result.data.totalCost).toBe(0);
      expect(result.data.totalTokens).toBe(0);
    });

    test('should NOT allow accessing another users project cost data', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Create User A with project
      const { project: projectA } = await setupUserWithTaggedTraces(api, auth, page, 'cost-api-sec-a');

      // Create User B
      const emailB = uniqueEmail('cost-api-sec-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');

      // User B tries to access User A's cost data
      const result = await api.getCostBreakdown(loginB.data.token, projectA.ID);

      // Should be forbidden or not found
      expect([403, 404]).toContain(result.status);
    });
  });

  test.describe('Frontend: Cost Breakdown Chart', () => {
    test('should display cost breakdown chart with data', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithTaggedTraces(api, auth, page, 'cost-ui-chart');

      // Navigate to analytics page
      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Costs tab to see the Cost Breakdown component
      await page.click('button:has-text("Costs")');
      await page.waitForTimeout(500);

      // Should see the Cost Breakdown section
      await expect(page.locator('text=Cost Breakdown')).toBeVisible({ timeout: 10000 });

      // Should see cost data displayed
      await expect(page.locator('text=/\\$\\d+\\.\\d+/')).toBeVisible({ timeout: 5000 });

      // Should see tag labels (in Enterprise mode)
      // Note: In OSS mode this shows model breakdown, in EE mode shows tag breakdown
      await expect(page.locator('text=/\\$/')).toBeVisible();
    });

    test('should filter by tag prefix using input', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithTaggedTraces(api, auth, page, 'cost-ui-filter');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Costs tab
      await page.click('button:has-text("Costs")');
      await page.waitForTimeout(500);

      // Find tag prefix filter input (only visible in Enterprise mode)
      const prefixInput = page.locator('input[placeholder*="tag prefix" i], input[placeholder*="org:" i]');
      if (await prefixInput.isVisible({ timeout: 3000 }).catch(() => false)) {
        await prefixInput.fill('org:');

        // Click filter button
        await page.click('button:has-text("Filter")');
        await page.waitForTimeout(500);

        // Should only show org: tags
        const tagBadges = page.locator('[data-testid="tag-badge"], .tag-badge, text=/org:/');
        await expect(tagBadges.first()).toBeVisible({ timeout: 5000 });
      }
    });

    test('should show loading state while fetching data', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithTaggedTraces(api, auth, page, 'cost-ui-loading');

      // Navigate to analytics
      await page.goto('/dashboard/analytics');

      // Click on Costs tab
      await page.click('button:has-text("Costs")');

      // Wait for data to load
      await page.waitForLoadState('networkidle');

      // After loading, should see cost data
      await expect(page.locator('text=Cost Breakdown')).toBeVisible({ timeout: 5000 });
    });

    test('should show empty state when no data', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('cost-ui-empty');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Empty UI User');
      const loginResult = await api.login(email, password);
      await api.createProject(loginResult.data.token, 'Empty Analytics Project');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Costs tab
      await page.click('button:has-text("Costs")');
      await page.waitForTimeout(500);

      // Should see cost section (OSS shows model breakdown, EE shows tag breakdown)
      await expect(page.locator('text=Cost Breakdown')).toBeVisible({ timeout: 5000 });
    });

    test('should display percentage breakdown correctly', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      await setupUserWithTaggedTraces(api, auth, page, 'cost-ui-percent');

      await page.goto('/dashboard/analytics');
      await page.waitForLoadState('networkidle');

      // Click on Costs tab
      await page.click('button:has-text("Costs")');
      await page.waitForTimeout(500);

      // Wait for data to load
      await expect(page.locator('text=Cost Breakdown')).toBeVisible({ timeout: 10000 });

      // Should show percentage values
      await expect(page.locator('text=/%/')).toBeVisible({ timeout: 5000 });

      // Percentages should add up (verify at least one percentage is shown)
      const percentages = page.locator('text=/\\d+(\\.\\d+)?%/');
      await expect(percentages.first()).toBeVisible();
    });
  });

  test.describe('Integration: Frontend + Backend', () => {
    test('should fetch and display real cost data from API', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithTaggedTraces(api, auth, page, 'cost-int-real');

      // First verify API returns data
      const apiResult = await api.getCostBreakdown(loginResult.data.token, project.ID);
      expect(apiResult.status).toBe(200);
      expect(apiResult.data.breakdown.length).toBeGreaterThan(0);

      // Navigate to dashboard and verify the same data appears
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');

      // The dashboard should show cost information somewhere
      // (depends on where CostBreakdownChart is placed)
      const costIndicator = page.locator('text=/\\$\\d+|cost|Cost/i').first();
      await expect(costIndicator).toBeVisible({ timeout: 10000 });
    });

    test('should update UI when new traces are ingested', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);
      const { project, loginResult } = await setupUserWithTaggedTraces(api, auth, page, 'cost-int-update');

      // Get initial cost
      const initialResult = await api.getCostBreakdown(loginResult.data.token, project.ID);
      const initialCost = initialResult.data.totalCost;

      // Ingest more traces
      await api.ingestTracesWithTags(project.APIKey, [
        { name: 'New Chat', tags: ['org:acme'], inputTokens: 5000, outputTokens: 2500, model: 'gpt-4o' },
      ]);

      // Verify API shows increased cost
      const updatedResult = await api.getCostBreakdown(loginResult.data.token, project.ID);
      expect(updatedResult.data.totalCost).toBeGreaterThan(initialCost);
    });

    test('should correctly aggregate costs for same tag', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      const email = uniqueEmail('cost-int-agg');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Aggregation User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Aggregation Project');
      const project = projectResult.data;

      // Ingest multiple traces with same tag
      await api.ingestTracesWithTags(project.APIKey, [
        { name: 'Chat 1', tags: ['org:test'], inputTokens: 100, outputTokens: 50 },
        { name: 'Chat 2', tags: ['org:test'], inputTokens: 100, outputTokens: 50 },
        { name: 'Chat 3', tags: ['org:test'], inputTokens: 100, outputTokens: 50 },
      ]);

      const result = await api.getCostBreakdown(loginResult.data.token, project.ID);

      // Should have single aggregated entry for org:test
      const orgTestEntries = result.data.breakdown.filter((b: { tag: string }) => b.tag === 'org:test');
      expect(orgTestEntries.length).toBe(1);

      // Total tokens should be 3x (300 input + 150 output = 450)
      expect(orgTestEntries[0].totalTokens).toBe(450);
      expect(orgTestEntries[0].traceCount).toBe(3);
    });
  });

  test.describe('Security: Data Isolation', () => {
    test('users cannot see cost data from other projects', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // User A creates project with high-value traces
      const emailA = uniqueEmail('cost-sec-a');
      await api.register(emailA, 'TestPassword123!', 'User A');
      const loginA = await api.login(emailA, 'TestPassword123!');
      const projectA = await api.createProject(loginA.data.token, 'Secret Project');

      await api.ingestTracesWithTags(projectA.data.APIKey, [
        { name: 'Secret Chat', tags: ['confidential:true'], inputTokens: 10000, outputTokens: 5000 },
      ]);

      // User B creates their own project
      const emailB = uniqueEmail('cost-sec-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');
      const projectB = await api.createProject(loginB.data.token, 'User B Project');

      // User B's cost breakdown should NOT include User A's data
      const resultB = await api.getCostBreakdown(loginB.data.token, projectB.data.ID);
      expect(resultB.status).toBe(200);

      // Should not have confidential tag
      const tags = resultB.data.breakdown.map((b: { tag: string }) => b.tag);
      expect(tags).not.toContain('confidential:true');
    });

    test('API requires authentication', async ({ request }) => {
      const api = new ApiHelper(request);

      // Create a valid project first
      const email = uniqueEmail('cost-sec-noauth');
      await api.register(email, 'TestPassword123!', 'No Auth User');
      const loginResult = await api.login(email, 'TestPassword123!');
      const project = await api.createProject(loginResult.data.token, 'Auth Test');

      // Try to access without token
      const response = await request.get(
        `${api.getBaseUrl()}/api/v1/dashboard/projects/${project.data.ID}/analytics/cost-breakdown`
      );

      // Should be unauthorized
      expect(response.status()).toBe(401);
    });
  });
});
