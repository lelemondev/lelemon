import { Page, BrowserContext } from '@playwright/test';

/**
 * Auth helper for E2E tests.
 * Handles login/logout through the UI.
 */
export class AuthHelper {
  private page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  /**
   * Login through the UI.
   */
  async login(email: string, password: string) {
    await this.page.goto('/login');

    // Fill login form
    await this.page.fill('input[type="email"], input[name="email"]', email);
    await this.page.fill('input[type="password"], input[name="password"]', password);

    // Submit
    await this.page.click('button[type="submit"]');

    // Wait for redirect to dashboard
    await this.page.waitForURL(/\/dashboard/, { timeout: 10000 });
  }

  /**
   * Register through the UI.
   */
  async register(email: string, password: string, name: string) {
    await this.page.goto('/signup');

    // Fill registration form
    await this.page.fill('input[name="name"]', name);
    await this.page.fill('input[type="email"], input[name="email"]', email);
    await this.page.fill('input[type="password"], input[name="password"]', password);

    // Submit
    await this.page.click('button[type="submit"]');

    // Wait for redirect
    await this.page.waitForURL(/\/dashboard|\/login/, { timeout: 10000 });
  }

  /**
   * Logout through the UI.
   */
  async logout() {
    // Click user menu or logout button
    const logoutButton = this.page.locator('button:has-text("Logout"), button:has-text("Sign out")');
    if (await logoutButton.isVisible()) {
      await logoutButton.click();
    } else {
      // Try opening user menu first
      await this.page.click('[data-testid="user-menu"], button:has-text("Account")');
      await this.page.click('button:has-text("Logout"), button:has-text("Sign out")');
    }

    // Wait for redirect to login
    await this.page.waitForURL(/\/login|\/$/);
  }

  /**
   * Check if user is logged in.
   */
  async isLoggedIn(): Promise<boolean> {
    try {
      // Check for dashboard elements or user menu
      return await this.page.locator('[data-testid="user-menu"], nav:has-text("Dashboard")').isVisible();
    } catch {
      return false;
    }
  }

  /**
   * Set auth token directly (for faster tests).
   * Must navigate to a page first to access localStorage.
   */
  async setToken(context: BrowserContext, token: string, user?: { id: string; email: string; name: string }) {
    // Navigate to login page first to get access to the origin's localStorage
    await this.page.goto('/login');

    // Set token and user in localStorage (matching auth-context.tsx keys)
    await this.page.evaluate(({ token, user }) => {
      localStorage.setItem('lelemon_token', token);
      if (user) {
        localStorage.setItem('lelemon_user', JSON.stringify({
          id: user.id,
          email: user.email,
          name: user.name,
          createdAt: new Date().toISOString(),
        }));
      }
    }, { token, user });

    // Also set as cookie for middleware/SSR
    await context.addCookies([
      {
        name: 'lelemon_token',
        value: token,
        domain: 'localhost',
        path: '/',
      },
    ]);
  }
}
