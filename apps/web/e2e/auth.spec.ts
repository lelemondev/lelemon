import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { uniqueEmail } from './fixtures/test-data';

/**
 * Authentication E2E tests.
 */
test.describe('Authentication', () => {
  test.describe('Registration', () => {
    test('should register a new user', async ({ page }) => {
      const email = uniqueEmail('register');
      const password = 'TestPassword123!';
      const name = 'New Test User';

      await page.goto('/signup');

      // Fill form using id selectors (matching the actual page)
      await page.fill('#name', name);
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.fill('#confirmPassword', password);

      // Submit
      await page.click('button[type="submit"]');

      // Should redirect to dashboard or login after successful registration
      await expect(page).toHaveURL(/\/dashboard|\/login/, { timeout: 15000 });
    });

    test('should show error for existing email', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('duplicate');

      // First, register via API
      await api.register(email, 'TestPassword123!', 'First User');

      // Try to register same email via UI
      await page.goto('/signup');
      await page.fill('#name', 'Second User');
      await page.fill('#email', email);
      await page.fill('#password', 'TestPassword123!');
      await page.fill('#confirmPassword', 'TestPassword123!');
      await page.click('button[type="submit"]');

      // Should show error (look for red error div)
      await expect(page.locator('.text-red-600, .text-red-500, [class*="error"]')).toBeVisible({ timeout: 5000 });
    });

    test('should show error for password mismatch', async ({ page }) => {
      await page.goto('/signup');
      await page.fill('#name', 'Test User');
      await page.fill('#email', uniqueEmail('mismatch'));
      await page.fill('#password', 'TestPassword123!');
      await page.fill('#confirmPassword', 'DifferentPassword!');
      await page.click('button[type="submit"]');

      // Should show password mismatch error
      await expect(page.locator('text=/do not match/i')).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Login', () => {
    test('should login with valid credentials', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('login');
      const password = 'TestPassword123!';

      // Create user first
      await api.register(email, password, 'Login Test User');

      // Login via UI
      await page.goto('/login');
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      // Should redirect to dashboard
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });
    });

    test('should show error for invalid credentials', async ({ page }) => {
      await page.goto('/login');
      await page.fill('#email', 'nonexistent@test.com');
      await page.fill('#password', 'wrongpassword');
      await page.click('button[type="submit"]');

      // Should show error (look for red error div)
      await expect(page.locator('.text-red-600, .text-red-500').first()).toBeVisible({ timeout: 5000 });
    });

    test('should show error for empty fields', async ({ page }) => {
      await page.goto('/login');

      // Try to submit empty form - browser validation should prevent
      await page.click('button[type="submit"]');

      // Should stay on login page
      await expect(page).toHaveURL(/\/login/);
    });
  });

  test.describe('Onboarding Flow', () => {
    test('new user should see welcome modal with API key after login', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('onboarding');
      const password = 'TestPassword123!';

      // Create user (but don't create any projects via API)
      await api.register(email, password, 'Onboarding Test User');

      // Login via UI
      await page.goto('/login');
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      // Should redirect to dashboard/config with welcome modal
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });

      // Should see the welcome modal with API key
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      await expect(welcomeModal).toBeVisible({ timeout: 10000 });

      // Should see the API key input (starts with le_)
      const apiKeyInput = page.locator('input[readonly]').first();
      await expect(apiKeyInput).toBeVisible();
      const apiKeyValue = await apiKeyInput.inputValue();
      expect(apiKeyValue).toMatch(/^le_[a-zA-Z0-9]+$/);

      // Should have a copy button
      await expect(page.getByRole('button', { name: 'Copy' })).toBeVisible();

      // Close the modal
      await page.click("button:has-text(\"Got it, let's go!\")");
      await expect(welcomeModal).not.toBeVisible({ timeout: 5000 });
    });

    test('existing user should skip onboarding and go directly to dashboard', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('existing');
      const password = 'TestPassword123!';

      // Create user AND project via API (simulating existing user)
      await api.register(email, password, 'Existing User');
      const loginResult = await api.login(email, password);
      await api.createProject(loginResult.data.token, 'Existing Project');

      // Now login via UI
      await page.goto('/login');
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      // Should redirect to dashboard
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });

      // Wait for page to stabilize
      await page.waitForLoadState('networkidle');

      // Should NOT see the welcome modal (user already has a project)
      const welcomeModal = page.locator('text=Welcome to Lelemon');
      await expect(welcomeModal).not.toBeVisible({ timeout: 3000 });
    });

    test('should remember last selected project after re-login', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('remember');
      const password = 'TestPassword123!';

      // Create user with multiple projects
      await api.register(email, password, 'Remember Project User');
      const loginResult = await api.login(email, password);
      await api.createProject(loginResult.data.token, 'Project Alpha');
      await api.createProject(loginResult.data.token, 'Project Beta');

      // Login via UI
      await page.goto('/login');
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      // Should redirect to dashboard
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });
      await page.waitForLoadState('networkidle');

      // Go to projects page
      await page.goto('/dashboard/projects');
      await page.waitForLoadState('networkidle');

      // Select "Project Beta" by clicking on the card in main content area (not sidebar)
      const projectBetaCard = page.locator('main').getByText('Project Beta');
      await projectBetaCard.click();

      // Wait for selection to be processed
      await page.waitForTimeout(500);

      // Verify it's selected by checking the sidebar project selector shows "Project Beta"
      // The combobox is in the sidebar (complementary role), not in nav
      await expect(page.locator('[role="combobox"]').getByText('Project Beta')).toBeVisible({ timeout: 5000 });

      // Also verify the card shows "Active" badge
      await expect(page.locator('main').getByText('Active')).toBeVisible();

      // Logout - click the button
      await page.click('button:has-text("Logout")');
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

      // Login again
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      // Should redirect to dashboard
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });
      await page.waitForLoadState('networkidle');

      // Go to projects page
      await page.goto('/dashboard/projects');
      await page.waitForLoadState('networkidle');

      // "Project Beta" should still be selected - check sidebar selector
      await expect(page.locator('[role="combobox"]').getByText('Project Beta')).toBeVisible({ timeout: 5000 });

      // And should have Active badge on the Project Beta card
      const betaCard = page.locator('main').locator('text=Project Beta').locator('..');
      await expect(betaCard.getByText('Active')).toBeVisible();
    });
  });

  test.describe('Protected Routes', () => {
    test('should redirect to login when not authenticated', async ({ page }) => {
      // Clear any existing auth
      await page.context().clearCookies();

      // Try to access protected route
      await page.goto('/dashboard/projects');

      // Should redirect to login
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    });

    test('should access dashboard when authenticated', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('protected');
      const password = 'TestPassword123!';

      // Create user
      await api.register(email, password, 'Protected Route User');

      // Login via UI to get proper auth state
      await page.goto('/login');
      await page.fill('#email', email);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      // Wait for redirect to dashboard
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });

      // Navigate to projects page
      await page.goto('/dashboard/projects');

      // Should stay on dashboard (not redirected to login)
      await expect(page).toHaveURL(/\/dashboard/);
    });
  });
});
