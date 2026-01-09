import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { AuthHelper } from './helpers/auth';
import { uniqueEmail } from './fixtures/test-data';

/**
 * Enterprise Edition (EE) Organization E2E tests.
 *
 * These tests verify multi-organization functionality:
 * - User info display in sidebar
 * - Organization switcher
 * - Role display per organization
 * - Switching between organizations
 * - Security boundaries between organizations
 * - Role-based access control
 *
 * SECURITY CRITICAL: These tests validate that:
 * - Users cannot access organizations they don't belong to
 * - Users cannot escalate their own privileges
 * - Users cannot see/modify data from other organizations
 * - Token manipulation doesn't grant unauthorized access
 *
 * NOTE: These tests are skipped until EE features are implemented.
 * They define the expected behavior for the EE organization system.
 */
test.describe('Enterprise: Organization', () => {
  test.describe('User Info', () => {
    test.skip('should show user name and email in sidebar', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-user');
      const password = 'TestPassword123!';
      const name = 'Enterprise User';

      // Create user and organization
      await api.register(email, password, name);
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Test Organization');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Should see user info section
      const userInfo = page.locator('[data-testid="user-info"]');
      await expect(userInfo).toBeVisible({ timeout: 5000 });

      // Should display user name
      await expect(userInfo.getByText(name)).toBeVisible();

      // Should display user email
      await expect(userInfo.getByText(email)).toBeVisible();
    });
  });

  test.describe('Organization Switcher', () => {
    test.skip('should show current organization with role', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-org');
      const password = 'TestPassword123!';

      // Create user and organization
      await api.register(email, password, 'Org Test User');
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Acme Corporation');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Should see organization switcher
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await expect(orgSwitcher).toBeVisible({ timeout: 5000 });

      // Should show organization name
      await expect(orgSwitcher.getByText('Acme Corporation')).toBeVisible();

      // Should show user's role (Owner for creator)
      await expect(orgSwitcher.getByText('Owner')).toBeVisible();
    });

    test.skip('should open dropdown with organization list', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-multi');
      const password = 'TestPassword123!';

      // Create user with multiple organizations
      await api.register(email, password, 'Multi Org User');
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Organization Alpha');
      await api.createOrganization(loginResult.data.token, 'Organization Beta');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Click organization switcher to open dropdown
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await orgSwitcher.click();

      // Should see dropdown with both organizations
      const dropdown = page.locator('[data-testid="organization-dropdown"]');
      await expect(dropdown).toBeVisible({ timeout: 3000 });

      // Should list all organizations
      await expect(dropdown.getByText('Organization Alpha')).toBeVisible();
      await expect(dropdown.getByText('Organization Beta')).toBeVisible();

      // Each should show the role
      await expect(dropdown.locator('text=Owner').first()).toBeVisible();
    });

    test.skip('should switch between organizations', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-switch');
      const password = 'TestPassword123!';

      // Create user with multiple organizations, each with a project
      await api.register(email, password, 'Switch Org User');
      const loginResult = await api.login(email, password);

      const org1 = await api.createOrganization(loginResult.data.token, 'First Org');
      const org2 = await api.createOrganization(loginResult.data.token, 'Second Org');

      // Create projects in each org (API would need org context)
      // This is a placeholder - actual implementation may differ

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Initially should be on first org
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await expect(orgSwitcher.getByText('First Org')).toBeVisible({ timeout: 5000 });

      // Open dropdown and switch to second org
      await orgSwitcher.click();
      await page.locator('[data-testid="organization-dropdown"]').getByText('Second Org').click();

      // Should now show second org
      await expect(orgSwitcher.getByText('Second Org')).toBeVisible({ timeout: 5000 });

      // Projects should update to second org's projects
      // (verification depends on implementation)
    });

    test.skip('should show different roles for different organizations', async ({ page, request }) => {
      // This test requires inviting user to another org with different role
      // Setup would be:
      // 1. User A creates Org Alpha (User A is Owner)
      // 2. User B creates Org Beta and invites User A as Member
      // 3. User A should see Owner in Org Alpha, Member in Org Beta

      const api = new ApiHelper(request);

      // Create first user who will own Org Alpha
      const emailA = uniqueEmail('ee-roles-a');
      const passwordA = 'TestPassword123!';
      await api.register(emailA, passwordA, 'User Alpha');
      const loginA = await api.login(emailA, passwordA);
      await api.createOrganization(loginA.data.token, 'Org Alpha');

      // Create second user who will own Org Beta and invite User A
      const emailB = uniqueEmail('ee-roles-b');
      const passwordB = 'TestPassword123!';
      await api.register(emailB, passwordB, 'User Beta');
      const loginB = await api.login(emailB, passwordB);
      const orgBeta = await api.createOrganization(loginB.data.token, 'Org Beta');

      // Invite User A to Org Beta as Member (requires invite API)
      // await api.inviteToOrganization(loginB.data.token, orgBeta.data.id, emailA, 'member');

      // User A accepts invitation (requires accept API)
      // await api.acceptInvitation(loginA.data.token, invitationId);

      // Login as User A and verify roles
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginA.data.token, loginA.data.user);

      await page.goto('/dashboard');

      // Open org switcher
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await orgSwitcher.click();

      const dropdown = page.locator('[data-testid="organization-dropdown"]');

      // Org Alpha should show Owner role
      const orgAlphaItem = dropdown.locator('[data-testid="org-item-org-alpha"]');
      await expect(orgAlphaItem.getByText('Owner')).toBeVisible();

      // Org Beta should show Member role
      const orgBetaItem = dropdown.locator('[data-testid="org-item-org-beta"]');
      await expect(orgBetaItem.getByText('Member')).toBeVisible();
    });

    test.skip('should show create organization option', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-create');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Create Org User');
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Existing Org');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Open org switcher
      await page.locator('[data-testid="organization-switcher"]').click();

      // Should see create organization option
      const createOrgOption = page.locator('[data-testid="create-organization"]');
      await expect(createOrgOption).toBeVisible();
      await expect(createOrgOption.getByText(/Create Organization|New Organization/i)).toBeVisible();
    });
  });

  test.describe('Organization Context', () => {
    test.skip('should persist selected organization after page reload', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-persist');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Persist Org User');
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Persistent Org');
      await api.createOrganization(loginResult.data.token, 'Another Org');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Switch to "Another Org"
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await orgSwitcher.click();
      await page.locator('[data-testid="organization-dropdown"]').getByText('Another Org').click();

      // Verify switch
      await expect(orgSwitcher.getByText('Another Org')).toBeVisible({ timeout: 5000 });

      // Reload page
      await page.reload();
      await page.waitForLoadState('networkidle');

      // Should still be on "Another Org"
      await expect(page.locator('[data-testid="organization-switcher"]').getByText('Another Org')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should update projects when organization changes', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-projects');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Projects Org User');
      const loginResult = await api.login(email, password);

      // Create orgs with different projects
      // Note: This requires org-scoped project creation API
      await api.createOrganization(loginResult.data.token, 'Org With Project A');
      await api.createOrganization(loginResult.data.token, 'Org With Project B');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Should see Project A in project selector initially
      // (depends on which org is default)

      // Switch organization
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await orgSwitcher.click();
      await page.locator('[data-testid="organization-dropdown"]').getByText('Org With Project B').click();

      // Project selector should now show Project B
      // (verification depends on implementation)
    });
  });

  test.describe('Role-based Access', () => {
    test.skip('should show Teams link for admins and owners', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-admin');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Admin User');
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Admin Org');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // As Owner, should see Teams in navigation
      await expect(page.locator('nav').getByText('Teams')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should hide Teams link for members and viewers', async ({ page, request }) => {
      // This requires:
      // 1. Owner creates org
      // 2. Owner invites user as Member/Viewer
      // 3. Member/Viewer logs in and shouldn't see Teams

      // Placeholder - actual test would need invitation flow
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-member');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Member User');
      // ... setup as member of org ...

      // As Member, should NOT see Teams in navigation
      // await expect(page.locator('nav').getByText('Teams')).not.toBeVisible();
    });

    test.skip('should show Billing link only for owners', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('ee-billing');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Billing User');
      const loginResult = await api.login(email, password);
      await api.createOrganization(loginResult.data.token, 'Billing Org');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // As Owner, should see Billing in navigation
      await expect(page.locator('nav').getByText('Billing')).toBeVisible({ timeout: 5000 });
    });
  });

  /**
   * SECURITY TESTS: Organization Isolation
   * These tests verify that users cannot access resources from organizations they don't belong to.
   */
  test.describe('Security: Organization Isolation', () => {
    test.skip('should NOT show organizations user is not a member of', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Create User A with Org A
      const emailA = uniqueEmail('sec-iso-a');
      await api.register(emailA, 'TestPassword123!', 'User A');
      const loginA = await api.login(emailA, 'TestPassword123!');
      await api.createOrganization(loginA.data.token, 'Secret Org A');

      // Create User B with Org B
      const emailB = uniqueEmail('sec-iso-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');
      await api.createOrganization(loginB.data.token, 'Org B');

      // Login as User B
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginB.data.token, loginB.data.user);

      await page.goto('/dashboard');

      // Open org switcher
      await page.locator('[data-testid="organization-switcher"]').click();

      // User B should NOT see User A's organization
      const dropdown = page.locator('[data-testid="organization-dropdown"]');
      await expect(dropdown.getByText('Secret Org A')).not.toBeVisible();

      // Should only see their own org
      await expect(dropdown.getByText('Org B')).toBeVisible();
    });

    test.skip('should NOT allow accessing organization by manipulating URL', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Create User A with Org A
      const emailA = uniqueEmail('sec-url-a');
      await api.register(emailA, 'TestPassword123!', 'User A');
      const loginA = await api.login(emailA, 'TestPassword123!');
      const orgA = await api.createOrganization(loginA.data.token, 'Protected Org');

      // Create User B
      const emailB = uniqueEmail('sec-url-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');
      await api.createOrganization(loginB.data.token, 'User B Org');

      // Login as User B
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginB.data.token, loginB.data.user);

      // Try to access User A's org settings directly via URL
      // This should redirect or show error, NOT show the org
      await page.goto(`/dashboard/organizations/${orgA.data?.id || 'fake-id'}/settings`);

      // Should see access denied or redirect
      await expect(page.locator('text=/access denied|not found|unauthorized|forbidden/i')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should NOT allow accessing projects from another organization', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Create User A with Org A and Project
      const emailA = uniqueEmail('sec-proj-a');
      await api.register(emailA, 'TestPassword123!', 'User A');
      const loginA = await api.login(emailA, 'TestPassword123!');
      await api.createOrganization(loginA.data.token, 'Org A');
      const projectA = await api.createProject(loginA.data.token, 'Secret Project');

      // Create User B
      const emailB = uniqueEmail('sec-proj-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');
      await api.createOrganization(loginB.data.token, 'Org B');

      // Login as User B
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginB.data.token, loginB.data.user);

      // Try to access User A's project traces directly
      await page.goto(`/dashboard/traces?projectId=${projectA.data?.id || 'fake-id'}`);

      // Should NOT see any traces from the other project
      // Should see empty state or error
      await expect(page.locator('text=/no traces|access denied|not found/i')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should NOT leak organization data in API responses', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Create two separate users with orgs
      const emailA = uniqueEmail('sec-leak-a');
      await api.register(emailA, 'TestPassword123!', 'User A');
      const loginA = await api.login(emailA, 'TestPassword123!');
      await api.createOrganization(loginA.data.token, 'Confidential Org');

      const emailB = uniqueEmail('sec-leak-b');
      await api.register(emailB, 'TestPassword123!', 'User B');
      const loginB = await api.login(emailB, 'TestPassword123!');

      // User B requests organizations list
      const orgsResponse = await request.get('/api/v1/organizations', {
        headers: { Authorization: `Bearer ${loginB.data.token}` },
      });

      const orgs = await orgsResponse.json();

      // Should not contain User A's organization
      const orgNames = orgs.data?.map((o: { name: string }) => o.name) || [];
      expect(orgNames).not.toContain('Confidential Org');
    });
  });

  /**
   * SECURITY TESTS: Role Enforcement
   * These tests verify that role-based permissions are properly enforced.
   */
  test.describe('Security: Role Enforcement', () => {
    test.skip('member should NOT be able to invite other users', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Owner creates org
      const ownerEmail = uniqueEmail('sec-role-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      const org = await api.createOrganization(ownerLogin.data.token, 'Role Test Org');

      // Owner invites Member
      const memberEmail = uniqueEmail('sec-role-member');
      await api.register(memberEmail, 'TestPassword123!', 'Member');
      // await api.inviteToOrganization(ownerLogin.data.token, org.data.id, memberEmail, 'member');

      const memberLogin = await api.login(memberEmail, 'TestPassword123!');

      // Login as Member
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), memberLogin.data.token, memberLogin.data.user);

      await page.goto('/dashboard/teams');

      // Member should NOT see invite button or it should be disabled
      const inviteButton = page.locator('button:has-text("Invite")');
      const isVisible = await inviteButton.isVisible().catch(() => false);

      if (isVisible) {
        // If button exists, it should be disabled
        await expect(inviteButton).toBeDisabled();
      }
    });

    test.skip('member should NOT be able to change other users roles', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Setup: Owner creates org, invites Admin and Member
      const ownerEmail = uniqueEmail('sec-role-change-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      await api.createOrganization(ownerLogin.data.token, 'Role Change Org');

      const memberEmail = uniqueEmail('sec-role-change-member');
      await api.register(memberEmail, 'TestPassword123!', 'Member');
      // Invite as member...

      const memberLogin = await api.login(memberEmail, 'TestPassword123!');

      // Login as Member
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), memberLogin.data.token, memberLogin.data.user);

      await page.goto('/dashboard/teams');

      // Member should NOT see role change dropdowns
      const roleDropdowns = page.locator('[data-testid="role-selector"]');
      await expect(roleDropdowns).toHaveCount(0);
    });

    test.skip('admin should NOT be able to change owner role', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Setup: Owner creates org, invites Admin
      const ownerEmail = uniqueEmail('sec-admin-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'The Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      await api.createOrganization(ownerLogin.data.token, 'Owner Protected Org');

      const adminEmail = uniqueEmail('sec-admin-admin');
      await api.register(adminEmail, 'TestPassword123!', 'Admin User');
      // Invite as admin...

      const adminLogin = await api.login(adminEmail, 'TestPassword123!');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), adminLogin.data.token, adminLogin.data.user);

      await page.goto('/dashboard/teams');

      // The owner's role dropdown should be disabled or not visible to admin
      const ownerRow = page.locator('[data-testid="member-row"]').filter({ hasText: 'The Owner' });
      const roleSelector = ownerRow.locator('[data-testid="role-selector"]');

      // Should either not exist or be disabled
      const isVisible = await roleSelector.isVisible().catch(() => false);
      if (isVisible) {
        await expect(roleSelector).toBeDisabled();
      }
    });

    test.skip('viewer should only have read access', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Setup viewer in org
      const ownerEmail = uniqueEmail('sec-viewer-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      await api.createOrganization(ownerLogin.data.token, 'Viewer Test Org');

      const viewerEmail = uniqueEmail('sec-viewer-viewer');
      await api.register(viewerEmail, 'TestPassword123!', 'Viewer');
      // Invite as viewer...

      const viewerLogin = await api.login(viewerEmail, 'TestPassword123!');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), viewerLogin.data.token, viewerLogin.data.user);

      await page.goto('/dashboard');

      // Viewer should NOT see: Teams, Billing, Create Project, Config edit buttons
      await expect(page.locator('nav').getByText('Teams')).not.toBeVisible();
      await expect(page.locator('nav').getByText('Billing')).not.toBeVisible();
      await expect(page.locator('button:has-text("New Project")')).not.toBeVisible();
    });

    test.skip('should NOT allow role escalation via API manipulation', async ({ request }) => {
      const api = new ApiHelper(request);

      // Member tries to promote themselves to admin via direct API call
      const ownerEmail = uniqueEmail('sec-esc-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      const org = await api.createOrganization(ownerLogin.data.token, 'Escalation Test Org');

      const memberEmail = uniqueEmail('sec-esc-member');
      await api.register(memberEmail, 'TestPassword123!', 'Sneaky Member');
      const memberLogin = await api.login(memberEmail, 'TestPassword123!');
      // Invited as member...

      // Member tries to change their own role
      const response = await request.patch(
        `/api/v1/organizations/${org.data?.id}/members/${memberLogin.data.user.id}`,
        {
          headers: { Authorization: `Bearer ${memberLogin.data.token}` },
          data: { role: 'owner' },
        }
      );

      // Should be forbidden
      expect(response.status()).toBe(403);
    });
  });

  /**
   * SECURITY TESTS: Token & Session
   * These tests verify token and session security.
   */
  test.describe('Security: Token & Session', () => {
    test.skip('should reject expired tokens', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('sec-expired');
      await api.register(email, 'TestPassword123!', 'Expired Token User');

      // Use a known expired token (would need to be crafted or use a very short expiry in test env)
      const expiredToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAwMDAwMDB9.fake';

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), expiredToken, { id: 'fake', email, name: 'Test' });

      await page.goto('/dashboard');

      // Should redirect to login
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    });

    test.skip('should reject tampered tokens', async ({ page }) => {
      // Use a tampered/invalid token
      const tamperedToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiJoYWNrZWQifQ.tampered';

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), tamperedToken, { id: 'hacked', email: 'hacker@test.com', name: 'Hacker' });

      await page.goto('/dashboard');

      // Should redirect to login
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    });

    test.skip('should invalidate session on logout across tabs', async ({ page, context, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('sec-logout');
      const password = 'TestPassword123!';

      await api.register(email, password, 'Logout Test User');
      const loginResult = await api.login(email, password);

      // Open two tabs with same session
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');
      const page2 = await context.newPage();
      await page2.goto('/dashboard');

      // Both should be on dashboard
      await expect(page).toHaveURL(/\/dashboard/);
      await expect(page2).toHaveURL(/\/dashboard/);

      // Logout from first tab
      await page.click('button:has-text("Logout")');
      await expect(page).toHaveURL(/\/login/);

      // Second tab should also be logged out on next navigation/refresh
      await page2.reload();
      await expect(page2).toHaveURL(/\/login/, { timeout: 10000 });
    });

    test.skip('should NOT allow localStorage manipulation to gain access', async ({ page }) => {
      // Manually set fake auth data in localStorage
      await page.goto('/login');
      await page.evaluate(() => {
        localStorage.setItem('lelemon_token', 'fake_token_12345');
        localStorage.setItem('lelemon_user', JSON.stringify({
          id: 'fake-user-id',
          email: 'fake@hacker.com',
          name: 'Fake User'
        }));
        localStorage.setItem('lelemon_current_org', 'fake-org-id');
      });

      await page.goto('/dashboard');

      // Should redirect to login because token validation fails on backend
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    });
  });

  /**
   * EDGE CASES
   * These tests verify correct behavior in unusual situations.
   */
  test.describe('Edge Cases', () => {
    test.skip('should handle user with no organizations gracefully', async ({ page, request }) => {
      const api = new ApiHelper(request);
      const email = uniqueEmail('edge-no-org');
      const password = 'TestPassword123!';

      // Register user but don't create any org (EE mode)
      await api.register(email, password, 'Orphan User');
      const loginResult = await api.login(email, password);

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Should see prompt to create or join an organization
      await expect(page.locator('text=/create.*organization|join.*organization|no organization/i')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should handle being removed from organization while logged in', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Owner creates org and invites member
      const ownerEmail = uniqueEmail('edge-remove-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      const org = await api.createOrganization(ownerLogin.data.token, 'Removal Test Org');

      const memberEmail = uniqueEmail('edge-remove-member');
      await api.register(memberEmail, 'TestPassword123!', 'Soon Removed');
      const memberLogin = await api.login(memberEmail, 'TestPassword123!');
      // Invite and accept...

      // Member is logged in and on dashboard
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), memberLogin.data.token, memberLogin.data.user);
      await page.goto('/dashboard');

      // Owner removes member via API (simulating another session)
      // await api.removeMember(ownerLogin.data.token, org.data.id, memberLogin.data.user.id);

      // Member tries to perform action or refresh
      await page.reload();

      // Should see error or be redirected to "no organization" state
      await expect(page.locator('text=/removed|no longer.*member|access.*revoked|no organization/i')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should handle organization being deleted while user is active', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Create org
      const ownerEmail = uniqueEmail('edge-delete-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      const org = await api.createOrganization(ownerLogin.data.token, 'To Be Deleted Org');

      // Invite another user
      const memberEmail = uniqueEmail('edge-delete-member');
      await api.register(memberEmail, 'TestPassword123!', 'Member');
      const memberLogin = await api.login(memberEmail, 'TestPassword123!');
      // Invite and accept...

      // Member is on dashboard
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), memberLogin.data.token, memberLogin.data.user);
      await page.goto('/dashboard');

      // Owner deletes org via API
      // await api.deleteOrganization(ownerLogin.data.token, org.data.id);

      // Member tries to do something
      await page.click('[data-testid="organization-switcher"]');

      // Should handle gracefully - show error or redirect
      await expect(page.locator('text=/deleted|not found|no longer exists/i')).toBeVisible({ timeout: 5000 });
    });

    test.skip('should handle pending invitation correctly', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Owner invites user
      const ownerEmail = uniqueEmail('edge-pending-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      await api.createOrganization(ownerLogin.data.token, 'Pending Invite Org');

      // Invitee registers but hasn't accepted
      const inviteeEmail = uniqueEmail('edge-pending-invitee');
      await api.register(inviteeEmail, 'TestPassword123!', 'Pending User');
      const inviteeLogin = await api.login(inviteeEmail, 'TestPassword123!');
      // Owner sends invite, but user hasn't accepted yet

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), inviteeLogin.data.token, inviteeLogin.data.user);

      await page.goto('/dashboard');

      // Should NOT see the org in switcher (invitation not accepted)
      // But might see invitation notification
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      if (await orgSwitcher.isVisible().catch(() => false)) {
        await orgSwitcher.click();
        await expect(page.locator('[data-testid="organization-dropdown"]').getByText('Pending Invite Org')).not.toBeVisible();
      }
    });

    test.skip('should prevent last owner from leaving organization', async ({ page, request }) => {
      const api = new ApiHelper(request);

      const ownerEmail = uniqueEmail('edge-last-owner');
      await api.register(ownerEmail, 'TestPassword123!', 'Last Owner');
      const ownerLogin = await api.login(ownerEmail, 'TestPassword123!');
      await api.createOrganization(ownerLogin.data.token, 'Single Owner Org');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), ownerLogin.data.token, ownerLogin.data.user);

      await page.goto('/dashboard/teams');

      // Try to leave organization or remove self
      // Should show error that last owner cannot leave
      const leaveButton = page.locator('button:has-text("Leave Organization")');
      if (await leaveButton.isVisible().catch(() => false)) {
        await leaveButton.click();

        // Should see error
        await expect(page.locator('text=/cannot leave|last owner|transfer ownership/i')).toBeVisible({ timeout: 5000 });
      }
    });

    test.skip('should handle concurrent organization switches', async ({ page, request }) => {
      const api = new ApiHelper(request);

      const email = uniqueEmail('edge-concurrent');
      await api.register(email, 'TestPassword123!', 'Concurrent User');
      const loginResult = await api.login(email, 'TestPassword123!');
      await api.createOrganization(loginResult.data.token, 'Org 1');
      await api.createOrganization(loginResult.data.token, 'Org 2');
      await api.createOrganization(loginResult.data.token, 'Org 3');

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), loginResult.data.token, loginResult.data.user);

      await page.goto('/dashboard');

      // Rapidly switch between orgs
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');

      await orgSwitcher.click();
      await page.locator('[data-testid="organization-dropdown"]').getByText('Org 2').click();

      // Immediately try to switch again
      await orgSwitcher.click();
      await page.locator('[data-testid="organization-dropdown"]').getByText('Org 3').click();

      // Should end up on Org 3 without errors
      await expect(orgSwitcher.getByText('Org 3')).toBeVisible({ timeout: 5000 });

      // No error toasts or UI glitches
      await expect(page.locator('.toast-error, [role="alert"][data-type="error"]')).not.toBeVisible();
    });
  });

  /**
   * REAL-WORLD SCENARIOS
   * These tests simulate realistic user workflows.
   */
  test.describe('Real-world Scenarios', () => {
    test.skip('complete onboarding: register -> create org -> invite team -> collaborate', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Step 1: New user registers
      const founderEmail = uniqueEmail('rw-founder');
      const password = 'TestPassword123!';
      await api.register(founderEmail, password, 'Startup Founder');

      // Step 2: Login and create organization
      await page.goto('/login');
      await page.fill('#email', founderEmail);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');

      await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 });

      // In EE mode, might need to create org first
      // Assuming redirected to org creation or dashboard

      // Step 3: Create organization if needed
      const createOrgButton = page.locator('button:has-text("Create Organization")');
      if (await createOrgButton.isVisible().catch(() => false)) {
        await createOrgButton.click();
        await page.fill('input[placeholder*="organization name" i]', 'My Startup');
        await page.click('button[type="submit"]:has-text("Create")');
      }

      // Step 4: Navigate to Teams and invite
      await page.click('nav >> text=Teams');
      await page.click('button:has-text("Invite")');

      // Fill invite form
      const teammateEmail = uniqueEmail('rw-teammate');
      await page.fill('input[type="email"]', teammateEmail);
      await page.locator('[data-testid="role-select"]').click();
      await page.locator('text=Admin').click();
      await page.click('button[type="submit"]:has-text("Send Invite")');

      // Should see success
      await expect(page.locator('text=/invitation sent|invited/i')).toBeVisible({ timeout: 5000 });
    });

    test.skip('consultant workflow: belong to multiple client orgs with different access', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Setup: Consultant belongs to 3 client orgs with different roles
      const consultantEmail = uniqueEmail('rw-consultant');
      await api.register(consultantEmail, 'TestPassword123!', 'Consultant');
      const consultantLogin = await api.login(consultantEmail, 'TestPassword123!');

      // Client 1: Owner (their own org)
      await api.createOrganization(consultantLogin.data.token, 'Consulting LLC');

      // Clients 2 & 3 would invite consultant as Admin/Viewer
      // (setup would require those orgs to exist and invite)

      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), consultantLogin.data.token, consultantLogin.data.user);

      await page.goto('/dashboard');

      // Verify can switch between orgs
      const orgSwitcher = page.locator('[data-testid="organization-switcher"]');
      await expect(orgSwitcher).toBeVisible({ timeout: 5000 });

      // Open and verify own org shows Owner
      await orgSwitcher.click();
      const dropdown = page.locator('[data-testid="organization-dropdown"]');
      await expect(dropdown.locator('text=Consulting LLC').locator('..').locator('text=Owner')).toBeVisible();
    });

    test.skip('agency handoff: transfer project ownership between orgs', async ({ page, request }) => {
      // Scenario: Agency completes project, client takes over

      const api = new ApiHelper(request);

      // Agency setup
      const agencyEmail = uniqueEmail('rw-agency');
      await api.register(agencyEmail, 'TestPassword123!', 'Agency Admin');
      const agencyLogin = await api.login(agencyEmail, 'TestPassword123!');
      await api.createOrganization(agencyLogin.data.token, 'Design Agency');

      // Client setup
      const clientEmail = uniqueEmail('rw-client');
      await api.register(clientEmail, 'TestPassword123!', 'Client CEO');
      const clientLogin = await api.login(clientEmail, 'TestPassword123!');
      await api.createOrganization(clientLogin.data.token, 'Client Corp');

      // Agency creates project for client
      await api.createProject(agencyLogin.data.token, 'Client Website');

      // Transfer would involve API for moving project between orgs
      // This tests that after transfer, client can see project and agency cannot

      // Login as client
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), clientLogin.data.token, clientLogin.data.user);

      await page.goto('/dashboard/projects');

      // After transfer, client should see the project
      // await expect(page.locator('text=Client Website')).toBeVisible();
    });

    test.skip('employee offboarding: removed user loses all access immediately', async ({ page, request }) => {
      const api = new ApiHelper(request);

      // Company setup
      const adminEmail = uniqueEmail('rw-admin');
      await api.register(adminEmail, 'TestPassword123!', 'IT Admin');
      const adminLogin = await api.login(adminEmail, 'TestPassword123!');
      const org = await api.createOrganization(adminLogin.data.token, 'BigCorp');

      // Employee setup
      const employeeEmail = uniqueEmail('rw-employee');
      await api.register(employeeEmail, 'TestPassword123!', 'Ex Employee');
      const employeeLogin = await api.login(employeeEmail, 'TestPassword123!');
      // Employee invited and accepted...

      // Employee is working
      const auth = new AuthHelper(page);
      await auth.setToken(page.context(), employeeLogin.data.token, employeeLogin.data.user);
      await page.goto('/dashboard');

      // Simulate: Admin removes employee (via API in another session)
      // await api.removeMember(adminLogin.data.token, org.data.id, employeeLogin.data.user.id);

      // Employee tries to access sensitive page
      await page.goto('/dashboard/config');

      // Should be denied
      await expect(page.locator('text=/access.*denied|removed|no longer.*access/i')).toBeVisible({ timeout: 5000 });
    });
  });
});
