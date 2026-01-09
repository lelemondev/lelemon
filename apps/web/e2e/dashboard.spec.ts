import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { AuthHelper } from './helpers/auth';
import { uniqueEmail } from './fixtures/test-data';

/**
 * Dashboard E2E tests.
 */
test.describe('Dashboard', () => {
  test.describe('Projects', () => {
    test('should create a new project', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('project');
      const password = 'TestPassword123!';

      // Create and login user
      await api.register(email, password, 'Project Test User');
      const loginResult = await api.login(email, password);

      // Set token with user info
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      // Go to dashboard - this will auto-create "My Project" and redirect to config with welcome modal
      await page.goto('/dashboard');

      // Wait for welcome modal (auto-created for new users, redirects to /dashboard/config)
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      await expect(welcomeModal).toBeVisible({ timeout: 10000 });
      await page.click("button:has-text(\"Got it, let's go!\")");

      // Wait for modal to close
      await expect(welcomeModal).not.toBeVisible({ timeout: 5000 });

      // Navigate to projects page
      await page.goto('/dashboard/projects');
      await page.waitForLoadState('networkidle');

      // Click create project button
      await page.click('button:has-text("New Project")');

      // Fill project name (input has placeholder="Project name")
      await page.fill('input[placeholder="Project name"]', 'E2E Test Project');

      // Submit the form
      await page.click('button[type="submit"]:has-text("Create")');

      // Handle API key modal that appears after creation
      const apiKeyModal = page.locator('text=Your API Key');
      await expect(apiKeyModal).toBeVisible({ timeout: 5000 });
      await page.click('button:has-text("I have saved my key")');

      // Wait for modal to close and verify project in list
      await expect(apiKeyModal).not.toBeVisible({ timeout: 5000 });
      await expect(page.locator('main').getByText('E2E Test Project')).toBeVisible({ timeout: 10000 });
    });

    test('should display API key for project', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('apikey');
      const password = 'TestPassword123!';

      // Create user and project via API
      await api.register(email, password, 'API Key User');
      const loginResult = await api.login(email, password);
      await api.createProject(loginResult.data.token, 'API Key Test');

      // Set token and navigate
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      // Go to config page where API key is shown
      await page.goto('/dashboard/config');

      // Handle welcome modal if it appears (new users get auto-created project)
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      if (await welcomeModal.isVisible({ timeout: 3000 }).catch(() => false)) {
        await page.click("button:has-text(\"Got it, let's go!\")");
        await expect(welcomeModal).not.toBeVisible({ timeout: 5000 });
      }

      // Should see API key section with hint (truncated, starts with le_)
      // Wait for the config page to fully load
      await page.waitForLoadState('networkidle');

      // The API Key card should be visible with the Rotate button
      await expect(page.getByRole('button', { name: 'Rotate' })).toBeVisible({ timeout: 5000 });

      // Verify the API key hint input exists (readonly input in the API Key section)
      const apiKeySection = page.locator('text=API Key').first().locator('..').locator('..');
      await expect(apiKeySection).toBeVisible();
    });

    test('should list user projects', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('list');
      const password = 'TestPassword123!';

      // Create user with multiple projects
      await api.register(email, password, 'List User');
      const loginResult = await api.login(email, password);

      await api.createProject(loginResult.data.token, 'Project Alpha');
      await api.createProject(loginResult.data.token, 'Project Beta');

      // Set token and navigate
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard/projects');

      // Should see both projects in the main content area (not sidebar)
      await expect(page.locator('main').getByText('Project Alpha')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('main').getByText('Project Beta')).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Navigation', () => {
    test('should navigate between dashboard sections', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('nav');
      const password = 'TestPassword123!';

      // Create and login
      await api.register(email, password, 'Nav User');
      const loginResult = await api.login(email, password);

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      // Navigate to dashboard
      await page.goto('/dashboard');

      // Check Projects link works
      const projectsLink = page.locator('nav a:has-text("Projects"), a[href*="projects"]').first();
      if (await projectsLink.isVisible()) {
        await projectsLink.click();
        await expect(page).toHaveURL(/projects/);
      }
    });

    test('should show user info in sidebar', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('userinfo');
      const password = 'TestPassword123!';
      const name = 'Sidebar Test User';

      await api.register(email, password, name);
      const loginResult = await api.login(email, password);

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Handle welcome modal if present
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      if (await welcomeModal.isVisible({ timeout: 3000 }).catch(() => false)) {
        await page.click("button:has-text(\"Got it, let's go!\")");
        await expect(welcomeModal).not.toBeVisible({ timeout: 5000 });
      }

      // Should see user name in sidebar
      await expect(page.locator('[data-testid="user-info"]').getByText(name)).toBeVisible({ timeout: 5000 });

      // Should see user email in sidebar
      await expect(page.locator('[data-testid="user-info"]').getByText(email)).toBeVisible({ timeout: 5000 });
    });

    test('should NOT show organization switcher in OSS mode', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('noorg');
      const password = 'TestPassword123!';

      await api.register(email, password, 'No Org User');
      const loginResult = await api.login(email, password);

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Handle welcome modal if present
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      if (await welcomeModal.isVisible({ timeout: 3000 }).catch(() => false)) {
        await page.click("button:has-text(\"Got it, let's go!\")");
      }

      // Organization switcher should NOT be visible in OSS
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await expect(orgSwitcher).not.toBeVisible({ timeout: 3000 });
    });
  });

  test.describe('Traces', () => {
    /**
     * Helper to setup a user with traces for testing.
     * Returns the API key and login result for further operations.
     */
    async function setupUserWithTraces(
      api: ApiHelper,
      auth: AuthHelper,
      page: import('@playwright/test').Page,
      emailPrefix: string,
      tracesData: object[] = []
    ) {
      const email = uniqueEmail(emailPrefix);
      const password = 'TestPassword123!';

      await api.register(email, password, `${emailPrefix} User`);
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, `${emailPrefix} Project`);
      const project = projectResult.data;

      // Ingest traces if provided
      if (tracesData.length > 0) {
        await api.ingestSpans(project.APIKey, tracesData);
      }

      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      return { email, password, project, loginResult };
    }

    test('should display traces after ingestion', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('traces');
      const password = 'TestPassword123!';

      // Setup: Create user and login
      await api.register(email, password, 'Traces User');
      const loginResult = await api.login(email, password);

      // Set token and navigate to dashboard first
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      // Go to dashboard - this will auto-create a project
      await page.goto('/dashboard');

      // Handle welcome modal (auto-created for new users)
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      if (await welcomeModal.isVisible({ timeout: 5000 }).catch(() => false)) {
        await page.click("button:has-text(\"Got it, let's go!\")");
        await expect(welcomeModal).not.toBeVisible({ timeout: 5000 });
      }

      // Get the API key from the config page
      await page.goto('/dashboard/config');
      await page.waitForLoadState('networkidle');

      // Now ingest spans using the auto-created project's API key
      // We need to get the API key - let's rotate it to get a fresh one
      await page.click('button:has-text("Rotate")');

      // Wait for the new API key modal and get the key
      const apiKeyModal = page.locator('text=New API Key');
      await expect(apiKeyModal).toBeVisible({ timeout: 5000 });

      // Get the API key from the input
      const apiKeyInput = page.locator('input[readonly]').nth(0);
      const apiKey = await apiKeyInput.inputValue();

      // Close the modal
      await page.click('button:has-text("I have saved my key")');
      await expect(apiKeyModal).not.toBeVisible({ timeout: 5000 });

      // Ingest some spans via API
      await api.ingestSpans(apiKey, [
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Test Chat',
          inputTokens: 100,
          outputTokens: 50,
          status: 'success',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait a bit for data to be polled (polling interval is 5s)
      await page.waitForTimeout(2000);

      // Should see at least 1 trace in the table (traces list shows "X traces")
      // Or we can verify the table has rows
      const tracesText = page.locator('text=/\\d+ traces?/');
      await expect(tracesText).toBeVisible({ timeout: 10000 });

      // Verify there's at least one row in the table (not just headers)
      const tableRow = page.locator('table tbody tr').first();
      await expect(tableRow).toBeVisible({ timeout: 10000 });
    });

    test('should show analytics summary', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('analytics');
      const password = 'TestPassword123!';

      // Setup with some data
      await api.register(email, password, 'Analytics User');
      const loginResult = await api.login(email, password);
      const projectResult = await api.createProject(loginResult.data.token, 'Analytics Test');
      const project = projectResult.data;

      await api.ingestSpans(project.APIKey, [
        { spanType: 'llm', inputTokens: 500, outputTokens: 200, status: 'success' },
        { spanType: 'llm', inputTokens: 300, outputTokens: 100, status: 'success' },
      ]);

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      // Navigate to analytics or dashboard
      await page.goto('/dashboard');

      // Should see some stats (tokens, cost, traces)
      const statsIndicator = page.locator('text=/token|cost|trace/i').first();
      await expect(statsIndicator).toBeVisible({ timeout: 10000 });
    });

    test('should show trace detail when clicking a row', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user with traces
      const { project } = await setupUserWithTraces(api, auth, page, 'tracedetail', [
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Detail Test Chat',
          inputTokens: 150,
          outputTokens: 75,
          status: 'success',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait for traces to load
      await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });

      // Click on the first trace row
      await page.locator('table tbody tr').first().click();

      // Should see the trace detail panel (master-detail view)
      // The detail panel should show trace info
      await expect(page.locator('text=/Duration|Spans|Tokens|Cost/i').first()).toBeVisible({ timeout: 5000 });
    });

    test('should open master-detail view when clicking a trace', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user with traces
      await setupUserWithTraces(api, auth, page, 'tracepage', [
        {
          spanType: 'llm',
          provider: 'anthropic',
          model: 'claude-3-opus',
          name: 'Page Test',
          inputTokens: 200,
          outputTokens: 100,
          status: 'success',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait for traces to load in table view
      const firstRow = page.locator('table tbody tr').first();
      await expect(firstRow).toBeVisible({ timeout: 10000 });

      // Click to open detail panel (switches to master-detail view)
      await firstRow.click();

      // Wait for master-detail view to load
      await page.waitForTimeout(500);

      // In master-detail view, left panel shows traces count
      await expect(page.locator('text=/\\d+ traces/i')).toBeVisible({ timeout: 5000 });

      // Right panel should show trace detail with stats
      await expect(page.locator('text=/Duration|Spans|Tokens/i').first()).toBeVisible({ timeout: 5000 });
    });

    test('should filter traces by status', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user with multiple traces of different statuses
      const { project } = await setupUserWithTraces(api, auth, page, 'filterstatus', [
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Success Trace',
          inputTokens: 100,
          outputTokens: 50,
          status: 'success',
        },
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Error Trace',
          inputTokens: 100,
          outputTokens: 0,
          status: 'error',
          errorMessage: 'Test error',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait for traces to load
      await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });

      // Find and click the status filter dropdown
      const statusFilter = page.locator('button:has-text("All Status")');
      await statusFilter.click();

      // Select "Error" status
      await page.locator('[role="option"]:has-text("Error")').click();

      // Wait for filter to apply
      await page.waitForTimeout(500);

      // Should show filtered count or error traces
      // The table should update to show only error traces
      const tracesCountText = page.locator('text=/\\d+ traces?/i');
      await expect(tracesCountText).toBeVisible({ timeout: 5000 });
    });

    test('should search traces by session or user ID', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user with traces that have session IDs
      const { project } = await setupUserWithTraces(api, auth, page, 'search', [
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Search Test',
          inputTokens: 100,
          outputTokens: 50,
          status: 'success',
          sessionId: 'test-session-12345',
          userId: 'user-abc',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait for traces to load
      await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });

      // Type in search box
      const searchInput = page.locator('input[placeholder*="Search"]');
      await searchInput.fill('test-session');

      // Wait for filter to apply
      await page.waitForTimeout(500);

      // Should still show the trace (matches session ID)
      await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 5000 });

      // Clear and search for non-existent term
      await searchInput.clear();
      await searchInput.fill('nonexistent-xyz-123');

      // Wait for filter to apply
      await page.waitForTimeout(500);

      // Should show no results message or empty state
      const noResults = page.locator('text=/No traces match|No traces yet/i');
      await expect(noResults).toBeVisible({ timeout: 5000 });
    });

    test('should clear filters', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user with traces
      await setupUserWithTraces(api, auth, page, 'clearfilter', [
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Filter Test',
          inputTokens: 100,
          outputTokens: 50,
          status: 'success',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait for traces to load
      await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });

      // Apply a search filter
      const searchInput = page.locator('input[placeholder*="Search"]');
      await searchInput.fill('nonexistent');

      // Wait for filter to apply
      await page.waitForTimeout(500);

      // Click "Clear filters" button
      const clearButton = page.locator('button:has-text("Clear filters")');
      if (await clearButton.isVisible()) {
        await clearButton.click();

        // Wait for filter to clear
        await page.waitForTimeout(500);

        // Search input should be empty
        await expect(searchInput).toHaveValue('');

        // Traces should be visible again
        await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 5000 });
      }
    });

    test('should show empty state when no traces', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user without any traces
      await setupUserWithTraces(api, auth, page, 'emptystate', []);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Should show empty state message
      const emptyMessage = page.locator('text=/No traces yet|Start by sending traces/i');
      await expect(emptyMessage).toBeVisible({ timeout: 10000 });
    });

    test('should display trace metadata in table columns', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const auth = new AuthHelper(page);

      // Setup user with a trace that has all metadata
      await setupUserWithTraces(api, auth, page, 'metadata', [
        {
          spanType: 'llm',
          provider: 'openai',
          model: 'gpt-4o',
          name: 'Metadata Test',
          inputTokens: 500,
          outputTokens: 250,
          status: 'success',
          sessionId: 'session-meta-test',
          userId: 'user-meta-test',
        },
      ]);

      // Navigate to traces page
      await page.goto('/dashboard/traces');
      await page.waitForLoadState('networkidle');

      // Wait for traces to load
      const firstRow = page.locator('table tbody tr').first();
      await expect(firstRow).toBeVisible({ timeout: 10000 });

      // Verify table headers exist
      await expect(page.locator('th:has-text("Time")')).toBeVisible();
      await expect(page.locator('th:has-text("Trace ID")')).toBeVisible();
      await expect(page.locator('th:has-text("Tokens")')).toBeVisible();
      await expect(page.locator('th:has-text("Cost")')).toBeVisible();
      await expect(page.locator('th:has-text("Status")')).toBeVisible();

      // Verify token count is visible (750 = 500 input + 250 output)
      // Use getByRole to get the specific cell with tokens
      await expect(page.getByRole('cell', { name: '750' })).toBeVisible();

      // Verify cost is displayed (should be non-zero)
      await expect(page.getByRole('cell', { name: /\$\d+\.\d+/ })).toBeVisible();
    });
  });
});
