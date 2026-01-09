import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';

test.describe('Security - Password Validation', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('rejects password shorter than 12 characters', async () => {
    const result = await api.register(
      `short-pass-${Date.now()}@test.com`,
      'Short1Pass!',  // 11 chars
      'Test User'
    );
    expect(result.status).toBe(400);
    expect(result.data?.error).toContain('12 characters');
  });

  test('rejects password without uppercase letter', async () => {
    const result = await api.register(
      `no-upper-${Date.now()}@test.com`,
      'lowercaseonly123',  // No uppercase
      'Test User'
    );
    expect(result.status).toBe(400);
    expect(result.data?.error).toContain('uppercase');
  });

  test('rejects password without lowercase letter', async () => {
    const result = await api.register(
      `no-lower-${Date.now()}@test.com`,
      'UPPERCASEONLY123',  // No lowercase
      'Test User'
    );
    expect(result.status).toBe(400);
    expect(result.data?.error).toContain('lowercase');
  });

  test('rejects password without number', async () => {
    const result = await api.register(
      `no-number-${Date.now()}@test.com`,
      'NoNumbersHere!',  // No digits
      'Test User'
    );
    expect(result.status).toBe(400);
    expect(result.data?.error).toContain('number');
  });

  test('accepts valid strong password', async () => {
    const result = await api.register(
      `valid-pass-${Date.now()}@test.com`,
      'ValidPass123!',  // 13 chars, upper, lower, number
      'Test User'
    );
    expect(result.status).toBe(201);
    expect(result.data?.token).toBeDefined();
  });

  test('accepts password with exactly 12 characters', async () => {
    const result = await api.register(
      `exact-12-${Date.now()}@test.com`,
      'Exactly12Pas',  // Exactly 12 chars
      'Test User'
    );
    expect(result.status).toBe(201);
    expect(result.data?.token).toBeDefined();
  });
});

test.describe('Security - Email Validation', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('rejects invalid email format - no @', async () => {
    const result = await api.register(
      'invalidemail.com',
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(400);
    expect(result.data?.error).toContain('email');
  });

  test('rejects invalid email format - no domain', async () => {
    const result = await api.register(
      'invalid@',
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(400);
    expect(result.data?.error).toContain('email');
  });

  test('normalizes email to lowercase', async () => {
    const email = `UPPERCASE-${Date.now()}@TEST.COM`;
    const result = await api.register(
      email,
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(201);

    // Try to login with lowercase version
    const loginResult = await api.login(email.toLowerCase(), 'ValidPass123!');
    expect(loginResult.status).toBe(200);
    expect(loginResult.data?.token).toBeDefined();
  });

  test('trims whitespace from email', async () => {
    const baseEmail = `whitespace-${Date.now()}@test.com`;
    const result = await api.register(
      `  ${baseEmail}  `,  // Extra whitespace
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(201);

    // Login with trimmed email should work
    const loginResult = await api.login(baseEmail, 'ValidPass123!');
    expect(loginResult.status).toBe(200);
  });

  test('accepts email with plus addressing', async () => {
    const result = await api.register(
      `user+tag-${Date.now()}@test.com`,
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(201);
    expect(result.data?.token).toBeDefined();
  });

  test('rejects empty email', async () => {
    const result = await api.register(
      '',
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(400);
  });

  test('rejects whitespace-only email', async () => {
    const result = await api.register(
      '   ',
      'ValidPass123!',
      'Test User'
    );
    expect(result.status).toBe(400);
  });
});

test.describe('Security - Security Headers', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('health endpoint returns security headers', async () => {
    const result = await api.getHeaders('/health');

    expect(result.headers['x-content-type-options']).toBe('nosniff');
    expect(result.headers['x-frame-options']).toBe('DENY');
    expect(result.headers['x-xss-protection']).toBe('1; mode=block');
    expect(result.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
    expect(result.headers['cache-control']).toBe('no-store');
  });

  test('API endpoint returns security headers', async () => {
    const result = await api.getHeaders('/api/v1/features');

    expect(result.headers['x-content-type-options']).toBe('nosniff');
    expect(result.headers['x-frame-options']).toBe('DENY');
  });

  test('authenticated endpoint returns security headers', async () => {
    // First register and login
    const email = `headers-test-${Date.now()}@test.com`;
    await api.register(email, 'ValidPass123!', 'Test User');
    const loginResult = await api.login(email, 'ValidPass123!');
    const token = loginResult.data?.token;

    const result = await api.getAuthenticatedHeaders('/api/v1/auth/me', token);

    expect(result.headers['x-content-type-options']).toBe('nosniff');
    expect(result.headers['x-frame-options']).toBe('DENY');
  });

  test('error responses include security headers', async () => {
    // Attempt login with invalid credentials
    const result = await api.loginRaw('nonexistent@test.com', 'wrongpassword');

    expect(result.headers['x-content-type-options']).toBe('nosniff');
    expect(result.headers['x-frame-options']).toBe('DENY');
  });
});

test.describe('Security - Authentication Edge Cases', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('login returns generic error for non-existent user', async () => {
    const result = await api.login(
      `nonexistent-${Date.now()}@test.com`,
      'ValidPass123!'
    );
    expect(result.status).toBe(401);
    // Should not reveal if email exists
    expect(result.data?.error).toBe('Invalid email or password');
  });

  test('login returns same error for wrong password', async () => {
    const email = `wrong-pass-${Date.now()}@test.com`;
    await api.register(email, 'ValidPass123!', 'Test User');

    const result = await api.login(email, 'WrongPass123!');
    expect(result.status).toBe(401);
    // Same generic error as non-existent user
    expect(result.data?.error).toBe('Invalid email or password');
  });

  test('rejects duplicate email registration', async () => {
    const email = `duplicate-${Date.now()}@test.com`;

    // First registration
    const first = await api.register(email, 'ValidPass123!', 'First User');
    expect(first.status).toBe(201);

    // Second registration with same email
    const second = await api.register(email, 'DifferentPass1!', 'Second User');
    expect(second.status).toBe(409);
    expect(second.data?.error).toContain('already');
  });

  test('rejects registration with empty name', async () => {
    const result = await api.register(
      `empty-name-${Date.now()}@test.com`,
      'ValidPass123!',
      ''
    );
    expect(result.status).toBe(400);
  });

  test('trims whitespace from name', async () => {
    const result = await api.register(
      `trimmed-name-${Date.now()}@test.com`,
      'ValidPass123!',
      '  Test User  '
    );
    expect(result.status).toBe(201);
    expect(result.data?.user?.name).toBe('Test User');
  });
});

test.describe('Security - Password Edge Cases', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('accepts password with unicode characters', async () => {
    const result = await api.register(
      `unicode-pass-${Date.now()}@test.com`,
      'ValidPass123!Ñoño',  // Unicode chars
      'Test User'
    );
    expect(result.status).toBe(201);
  });

  test('accepts password with emojis', async () => {
    const result = await api.register(
      `emoji-pass-${Date.now()}@test.com`,
      'ValidPass123!🔐',  // Emoji
      'Test User'
    );
    expect(result.status).toBe(201);
  });

  test('accepts very long password', async () => {
    const longPassword = 'ValidPass1!' + 'a'.repeat(200);
    const result = await api.register(
      `long-pass-${Date.now()}@test.com`,
      longPassword,
      'Test User'
    );
    expect(result.status).toBe(201);
  });

  test('password with only spaces should fail', async () => {
    const result = await api.register(
      `spaces-pass-${Date.now()}@test.com`,
      '            ',  // 12 spaces
      'Test User'
    );
    expect(result.status).toBe(400);
  });

  test('password with mixed unicode uppercase/lowercase', async () => {
    const result = await api.register(
      `unicode-case-${Date.now()}@test.com`,
      'Пароль123Тест',  // Cyrillic with upper and lower
      'Test User'
    );
    // This might pass or fail depending on implementation
    // The important thing is it doesn't crash
    expect([200, 201, 400]).toContain(result.status);
  });
});

test.describe('Security - Input Validation Edge Cases', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('handles very long email gracefully', async () => {
    const longEmail = 'a'.repeat(200) + '@test.com';
    const result = await api.register(
      longEmail,
      'ValidPass123!',
      'Test User'
    );
    // Should either accept (if valid) or reject with proper error, not crash
    expect([201, 400]).toContain(result.status);
  });

  test('handles very long name gracefully', async () => {
    const longName = 'A'.repeat(1000);
    const result = await api.register(
      `long-name-${Date.now()}@test.com`,
      'ValidPass123!',
      longName
    );
    // Should either accept or reject gracefully
    expect([201, 400]).toContain(result.status);
  });

  test('handles special characters in name', async () => {
    const result = await api.register(
      `special-name-${Date.now()}@test.com`,
      'ValidPass123!',
      "O'Brien-Smith <script>alert('xss')</script>"
    );
    expect(result.status).toBe(201);
    // Name should be stored but any output should be escaped
    expect(result.data?.user?.name).not.toContain('<script>');
  });

  test('handles null bytes in input', async () => {
    const result = await api.register(
      `null-byte-${Date.now()}@test.com`,
      'ValidPass123!\x00injection',
      'Test User'
    );
    // Should handle gracefully
    expect([201, 400]).toContain(result.status);
  });

  test('handles JSON injection attempt', async () => {
    const result = await api.register(
      `json-inj-${Date.now()}@test.com`,
      'ValidPass123!","admin":true,"password":"',
      'Test User'
    );
    // Should either work or fail validation, not allow injection
    expect([201, 400]).toContain(result.status);
    if (result.status === 201) {
      expect(result.data?.user?.admin).toBeUndefined();
    }
  });
});

test.describe('Security - Rate Limiting', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('allows normal login attempts', async () => {
    const email = `rate-limit-${Date.now()}@test.com`;
    await api.register(email, 'ValidPass123!', 'Test User');

    // A few attempts should be fine
    for (let i = 0; i < 3; i++) {
      const result = await api.login(email, 'WrongPass123!');
      expect(result.status).toBe(401); // Wrong password, but not rate limited
    }
  });

  // Note: This test is marked as slow because it tests rate limiting
  // In a real scenario, you'd need to make many requests quickly
  test.skip('returns 429 after too many failed attempts', async () => {
    // This test is skipped by default because:
    // 1. Rate limiting is 10 req/min, testing would slow down the suite
    // 2. IP-based rate limiting may be affected by test infrastructure
    // To test manually: make 11+ requests in rapid succession
    const email = `rate-limit-test-${Date.now()}@test.com`;

    let rateLimited = false;
    for (let i = 0; i < 15 && !rateLimited; i++) {
      const result = await api.login(email, 'WrongPass123!');
      if (result.status === 429) {
        rateLimited = true;
        expect(result.data?.error).toContain('Too many requests');
      }
    }
    expect(rateLimited).toBe(true);
  });
});

test.describe('Security - JWT Token', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('JWT token is returned on successful login', async () => {
    const email = `jwt-test-${Date.now()}@test.com`;
    await api.register(email, 'ValidPass123!', 'Test User');

    const result = await api.login(email, 'ValidPass123!');
    expect(result.status).toBe(200);
    expect(result.data?.token).toBeDefined();
    expect(result.data?.token).toMatch(/^[\w-]+\.[\w-]+\.[\w-]+$/); // JWT format
  });

  test('protected endpoint rejects missing token', async () => {
    const result = await api.getHeaders('/api/v1/auth/me');
    expect(result.status).toBe(401);
  });

  test('protected endpoint rejects invalid token', async () => {
    const result = await api.getAuthenticatedHeaders(
      '/api/v1/auth/me',
      'invalid.token.here'
    );
    expect(result.status).toBe(401);
  });

  test('protected endpoint rejects malformed token', async () => {
    const result = await api.getAuthenticatedHeaders(
      '/api/v1/auth/me',
      'not-even-a-jwt'
    );
    expect(result.status).toBe(401);
  });

  test('protected endpoint accepts valid token', async () => {
    const email = `jwt-valid-${Date.now()}@test.com`;
    await api.register(email, 'ValidPass123!', 'Test User');
    const loginResult = await api.login(email, 'ValidPass123!');

    const result = await api.getAuthenticatedHeaders(
      '/api/v1/auth/me',
      loginResult.data?.token
    );
    expect(result.status).toBe(200);
  });
});

test.describe('Security - OAuth Edge Cases', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('OAuth endpoint returns 501 when not configured', async ({ request }) => {
    // This test checks behavior when OAuth is not configured
    // If OAuth IS configured, we skip this test
    const isConfigured = await api.isOAuthConfigured();
    test.skip(isConfigured, 'OAuth is configured, skipping not-configured test');

    const result = await api.initiateGoogleOAuth();
    expect(result.status).toBe(501);
  });

  test('OAuth initiation sets state cookie and redirects', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result = await api.initiateGoogleOAuth();

    // Should redirect to Google
    expect(result.status).toBe(307);
    expect(result.location).toContain('accounts.google.com');
    expect(result.location).toContain('state=');

    // Should set oauth_state cookie
    const setCookie = result.headers['set-cookie'];
    expect(setCookie).toBeDefined();
    expect(setCookie).toContain('oauth_state=');
    expect(setCookie).toContain('HttpOnly');
  });

  test('callback rejects missing state parameter', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result = await api.oauthCallback({
      code: 'fake-code',
      // No state parameter
      cookie: 'oauth_state=some-state-value',
    });

    // Should redirect to login with error
    expect(result.status).toBe(307);
    expect(result.location).toContain('/login');
    expect(result.location).toContain('error=invalid_state');
  });

  test('callback rejects mismatched state', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result = await api.oauthCallback({
      code: 'fake-code',
      state: 'state-from-url',
      cookie: 'oauth_state=different-state-in-cookie',
    });

    // Should redirect to login with error
    expect(result.status).toBe(307);
    expect(result.location).toContain('/login');
    expect(result.location).toContain('error=invalid_state');
  });

  test('callback rejects missing cookie', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result = await api.oauthCallback({
      code: 'fake-code',
      state: 'some-state',
      // No cookie
    });

    // Should redirect to login with error
    expect(result.status).toBe(307);
    expect(result.location).toContain('/login');
    expect(result.location).toContain('error=invalid_state');
  });

  test('callback rejects missing code', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const matchingState = 'matching-state-value';
    const result = await api.oauthCallback({
      // No code
      state: matchingState,
      cookie: `oauth_state=${matchingState}`,
    });

    // Should redirect to login with error
    expect(result.status).toBe(307);
    expect(result.location).toContain('/login');
    expect(result.location).toContain('error=no_code');
  });

  test('callback handles error parameter from Google', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const matchingState = 'matching-state-value';
    const result = await api.oauthCallback({
      error: 'access_denied',
      state: matchingState,
      cookie: `oauth_state=${matchingState}`,
    });

    // Should redirect to login with the error from Google
    expect(result.status).toBe(307);
    expect(result.location).toContain('/login');
    expect(result.location).toContain('error=access_denied');
  });

  test('callback rejects invalid/expired code', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const matchingState = 'matching-state-value';
    const result = await api.oauthCallback({
      code: 'invalid-or-expired-code',
      state: matchingState,
      cookie: `oauth_state=${matchingState}`,
    });

    // Should redirect to login with auth_failed error
    // (Google will reject the invalid code)
    expect(result.status).toBe(307);
    expect(result.location).toContain('/login');
    expect(result.location).toContain('error=auth_failed');
  });

  test('callback clears state cookie after use', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const matchingState = 'matching-state-value';
    const result = await api.oauthCallback({
      code: 'some-code',
      state: matchingState,
      cookie: `oauth_state=${matchingState}`,
    });

    // Should set cookie with MaxAge=-1 to delete it
    const setCookie = result.headers['set-cookie'];
    expect(setCookie).toBeDefined();
    expect(setCookie).toContain('oauth_state=');
    expect(setCookie).toContain('Max-Age=0');
  });

  test('state parameter is URL-safe base64', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result = await api.initiateGoogleOAuth();

    // Extract state from redirect URL
    const url = new URL(result.location || '');
    const state = url.searchParams.get('state');

    expect(state).toBeDefined();
    // URL-safe base64: only alphanumeric, -, _, =
    expect(state).toMatch(/^[A-Za-z0-9_=-]+$/);
    // Should be ~43 chars (32 bytes base64 encoded)
    expect(state!.length).toBeGreaterThanOrEqual(40);
  });

  test('state is unique per request', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result1 = await api.initiateGoogleOAuth();
    const result2 = await api.initiateGoogleOAuth();

    const url1 = new URL(result1.location || '');
    const url2 = new URL(result2.location || '');

    const state1 = url1.searchParams.get('state');
    const state2 = url2.searchParams.get('state');

    expect(state1).not.toBe(state2);
  });
});

test.describe('Security - OAuth CSRF Protection', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('prevents CSRF by requiring state match', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    // Attacker tries to use their own state with victim's cookie
    const attackerState = 'attacker-controlled-state';
    const victimCookie = 'oauth_state=victim-state-from-cookie';

    const result = await api.oauthCallback({
      code: 'some-authorization-code',
      state: attackerState,
      cookie: victimCookie,
    });

    // Should reject - states don't match
    expect(result.status).toBe(307);
    expect(result.location).toContain('error=invalid_state');
  });

  test('state cookie has correct security attributes', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    const result = await api.initiateGoogleOAuth();
    const setCookie = result.headers['set-cookie'];

    // HttpOnly prevents JavaScript access
    expect(setCookie).toContain('HttpOnly');

    // SameSite=Lax allows redirect from Google but prevents CSRF
    expect(setCookie).toMatch(/SameSite=Lax/i);

    // Path should be / for the callback to access it
    expect(setCookie).toContain('Path=/');

    // Max-Age should be reasonable (600 = 10 minutes)
    expect(setCookie).toMatch(/Max-Age=\d+/);
  });

  test('expired state cookie is rejected', async () => {
    const isConfigured = await api.isOAuthConfigured();
    test.skip(!isConfigured, 'OAuth not configured');

    // Simulate an expired cookie scenario by not providing the cookie
    // (browser would not send expired cookies)
    const result = await api.oauthCallback({
      code: 'valid-looking-code',
      state: 'some-state',
      // No cookie - simulates expired
    });

    expect(result.status).toBe(307);
    expect(result.location).toContain('error=invalid_state');
  });
});
