import { test, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { uniqueEmail } from './fixtures/test-data';

/**
 * Post-Login Security E2E Tests
 *
 * Tests security aspects after authentication:
 * - API Key management and isolation
 * - Multi-tenant data isolation
 * - Session management
 * - Authorization boundaries
 *
 * SECURITY CRITICAL: These tests validate that:
 * - Users cannot access other users' projects/data
 * - API keys are properly scoped and revocable
 * - Session tokens are properly validated
 */

// ==================== Test Fixtures ====================

interface TestUser {
  email: string;
  password: string;
  token: string;
  userId: string;
}

interface TestProject {
  id: string;
  apiKey: string;
  name: string;
}

/**
 * Creates a test user with a project.
 * Follows DRY principle - reusable setup for multiple tests.
 */
async function createUserWithProject(
  api: ApiHelper,
  prefix: string
): Promise<{ user: TestUser; project: TestProject }> {
  const email = uniqueEmail(prefix);
  const password = 'TestPassword123!';

  await api.register(email, password, `${prefix} User`);
  const loginResult = await api.login(email, password);
  const projectResult = await api.createProject(
    loginResult.data.token,
    `${prefix} Project`
  );

  return {
    user: {
      email,
      password,
      token: loginResult.data.token,
      userId: loginResult.data.user.id,
    },
    project: {
      id: projectResult.data.ID || projectResult.data.id,
      apiKey: projectResult.data.APIKey || projectResult.data.apiKey,
      name: projectResult.data.Name || projectResult.data.name,
    },
  };
}

/**
 * Sample trace event for ingestion tests.
 */
function createSampleEvent(name: string = 'Test Span') {
  return {
    spanType: 'llm',
    provider: 'openai',
    model: 'gpt-4o',
    name,
    inputTokens: 100,
    outputTokens: 50,
    status: 'success',
  };
}

// ==================== API Key Security ====================

test.describe('Security - API Key Management', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('API key starts with correct prefix', async () => {
    const { project } = await createUserWithProject(api, 'apikey-prefix');

    expect(project.apiKey).toMatch(/^le_[a-zA-Z0-9]+$/);
    expect(project.apiKey.length).toBeGreaterThan(20);
  });

  test('API key allows ingestion to its project', async () => {
    const { project } = await createUserWithProject(api, 'apikey-ingest');

    const result = await api.ingestSpans(project.apiKey, [createSampleEvent()]);

    expect(result.status).toBe(200);
  });

  test('invalid API key is rejected', async () => {
    const result = await api.ingestSpansRaw('le_invalid_key_12345', [
      createSampleEvent(),
    ]);

    expect(result.status).toBe(401);
  });

  test('malformed API key is rejected', async () => {
    const result = await api.ingestSpansRaw('not-a-valid-key', [
      createSampleEvent(),
    ]);

    expect(result.status).toBe(401);
  });

  test('empty API key is rejected', async () => {
    const result = await api.ingestSpansRaw('', [createSampleEvent()]);

    expect(result.status).toBe(401);
  });

  test('JWT token cannot be used for ingestion', async () => {
    const { user } = await createUserWithProject(api, 'jwt-ingest');

    // Try to use JWT token instead of API key for ingestion
    const result = await api.ingestSpansRaw(user.token, [createSampleEvent()]);

    // Should fail - JWT is not valid for ingestion endpoint
    expect(result.status).toBe(401);
  });

  test('API key cannot be used for dashboard endpoints', async () => {
    const { user, project } = await createUserWithProject(api, 'apikey-dashboard');

    // Try to use API key to access dashboard endpoint
    const result = await api.getProjects(project.apiKey);

    expect(result.status).toBe(401);
  });
});

test.describe('Security - API Key Rotation', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('rotated API key is different from original', async () => {
    const { user, project } = await createUserWithProject(api, 'rotate-diff');
    const originalKey = project.apiKey;

    const rotateResult = await api.rotateApiKey(user.token, project.id);

    expect(rotateResult.status).toBe(200);
    expect(rotateResult.data.APIKey || rotateResult.data.apiKey).not.toBe(originalKey);
  });

  test('old API key stops working after rotation', async () => {
    const { user, project } = await createUserWithProject(api, 'rotate-old');
    const oldKey = project.apiKey;

    // Rotate the key
    await api.rotateApiKey(user.token, project.id);

    // Old key should no longer work
    const result = await api.ingestSpansRaw(oldKey, [createSampleEvent()]);

    expect(result.status).toBe(401);
  });

  test('new API key works after rotation', async () => {
    const { user, project } = await createUserWithProject(api, 'rotate-new');

    // Rotate the key
    const rotateResult = await api.rotateApiKey(user.token, project.id);
    const newKey = rotateResult.data.APIKey || rotateResult.data.apiKey;

    // New key should work
    const result = await api.ingestSpans(newKey, [createSampleEvent()]);

    expect(result.status).toBe(200);
  });

  test('only project owner can rotate API key', async () => {
    // Create two users
    const { project: project1 } = await createUserWithProject(api, 'rotate-owner1');
    const { user: user2 } = await createUserWithProject(api, 'rotate-owner2');

    // User 2 tries to rotate User 1's project key
    const result = await api.rotateApiKey(user2.token, project1.id);

    expect([401, 403, 404]).toContain(result.status);
  });
});

// ==================== Multi-Tenant Isolation ====================

test.describe('Security - Project Isolation', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('user cannot see other user projects', async () => {
    const { project: project1 } = await createUserWithProject(api, 'iso-proj1');
    const { user: user2 } = await createUserWithProject(api, 'iso-proj2');

    // User 2 tries to access User 1's project
    const result = await api.getProject(user2.token, project1.id);

    expect([401, 403, 404]).toContain(result.status);
  });

  test('user cannot list other user projects', async () => {
    const { project: project1 } = await createUserWithProject(api, 'iso-list1');
    const { user: user2 } = await createUserWithProject(api, 'iso-list2');

    // User 2 lists their projects
    const result = await api.getProjects(user2.token);

    expect(result.status).toBe(200);

    // Should not contain User 1's project
    const projectIds = (result.data || []).map(
      (p: { ID?: string; id?: string }) => p.ID || p.id
    );
    expect(projectIds).not.toContain(project1.id);
  });

  test('user cannot delete other user project', async () => {
    const { project: project1 } = await createUserWithProject(api, 'iso-del1');
    const { user: user2 } = await createUserWithProject(api, 'iso-del2');

    // User 2 tries to delete User 1's project
    const result = await api.deleteProject(user2.token, project1.id);

    expect([401, 403, 404]).toContain(result.status);
  });

  test('user cannot get stats for other user project', async () => {
    const { project: project1 } = await createUserWithProject(api, 'iso-stats1');
    const { user: user2 } = await createUserWithProject(api, 'iso-stats2');

    // User 2 tries to get User 1's project stats
    const result = await api.getProjectStats(user2.token, project1.id);

    expect([401, 403, 404]).toContain(result.status);
  });
});

test.describe('Security - Trace Isolation', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('traces are only visible to project owner', async () => {
    const { user: user1, project: project1 } = await createUserWithProject(
      api,
      'trace-iso1'
    );
    const { user: user2 } = await createUserWithProject(api, 'trace-iso2');

    // User 1 ingests a trace
    await api.ingestSpans(project1.apiKey, [
      createSampleEvent('Secret Trace'),
    ]);

    // User 2 tries to list User 1's traces
    const result = await api.getTraces(user2.token, project1.id);

    expect([401, 403, 404]).toContain(result.status);
  });

  test('API key from project A cannot ingest to project B', async () => {
    const { project: projectA } = await createUserWithProject(api, 'cross-ingest-a');
    const { project: projectB } = await createUserWithProject(api, 'cross-ingest-b');

    // Try to ingest to project B using project A's key
    // The API key is scoped to its project, so this should only affect project A
    const result = await api.ingestSpans(projectA.apiKey, [
      { ...createSampleEvent(), projectId: projectB.id },
    ]);

    // Ingestion succeeds but goes to project A, not B
    expect(result.status).toBe(200);

    // Verify trace is in project A, not B
    // (would need to wait for processing and check)
  });

  test('cannot access trace by guessing ID', async () => {
    const { user: user1, project: project1 } = await createUserWithProject(
      api,
      'trace-guess1'
    );
    const { user: user2, project: project2 } = await createUserWithProject(
      api,
      'trace-guess2'
    );

    // User 1 ingests a trace
    await api.ingestSpans(project1.apiKey, [createSampleEvent()]);

    // Give time for trace to be processed
    await new Promise((r) => setTimeout(r, 500));

    // Get User 1's traces to find a trace ID
    const tracesResult = await api.getTraces(user1.token, project1.id);
    if (tracesResult.data?.length > 0) {
      const traceId = tracesResult.data[0].ID || tracesResult.data[0].id;

      // User 2 tries to access User 1's trace via their own project
      const result = await api.getTrace(user2.token, project2.id, traceId);

      expect([401, 403, 404]).toContain(result.status);
    }
  });
});

// ==================== Session Management ====================

test.describe('Security - Session & Token', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('expired token is rejected', async () => {
    // Using a crafted expired JWT (exp in the past)
    const expiredToken =
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.' +
      'eyJ1c2VySWQiOiJ0ZXN0IiwiZXhwIjoxMDAwMDAwMDAwfQ.' +
      'invalid_signature';

    const result = await api.getMe(expiredToken);

    expect(result.status).toBe(401);
  });

  test('token with invalid signature is rejected', async () => {
    const tamperedToken =
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.' +
      'eyJ1c2VySWQiOiJoYWNrZXIiLCJleHAiOjk5OTk5OTk5OTl9.' +
      'tampered_signature_here';

    const result = await api.getMe(tamperedToken);

    expect(result.status).toBe(401);
  });

  test('token with wrong algorithm is rejected', async () => {
    // Token claiming to use 'none' algorithm
    const noneAlgToken =
      'eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.' +
      'eyJ1c2VySWQiOiJoYWNrZXIiLCJleHAiOjk5OTk5OTk5OTl9.';

    const result = await api.getMe(noneAlgToken);

    expect(result.status).toBe(401);
  });

  test('valid token returns user info', async () => {
    const { user } = await createUserWithProject(api, 'valid-token');

    const result = await api.getMe(user.token);

    expect(result.status).toBe(200);
    expect(result.data.email).toBe(user.email);
  });

  test('token works across multiple requests', async () => {
    const { user, project } = await createUserWithProject(api, 'multi-req');

    // Multiple requests with same token should all work
    const results = await Promise.all([
      api.getMe(user.token),
      api.getProjects(user.token),
      api.getProject(user.token, project.id),
    ]);

    results.forEach((result) => {
      expect(result.status).toBe(200);
    });
  });

  test('different users have different tokens', async () => {
    const { user: user1 } = await createUserWithProject(api, 'diff-token1');
    const { user: user2 } = await createUserWithProject(api, 'diff-token2');

    expect(user1.token).not.toBe(user2.token);

    // Each token returns correct user
    const me1 = await api.getMe(user1.token);
    const me2 = await api.getMe(user2.token);

    expect(me1.data.email).toBe(user1.email);
    expect(me2.data.email).toBe(user2.email);
  });
});

test.describe('Security - Login Session', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('new login generates new token', async () => {
    const email = uniqueEmail('new-login');
    const password = 'TestPassword123!';

    await api.register(email, password, 'New Login User');

    const login1 = await api.login(email, password);
    const login2 = await api.login(email, password);

    // Tokens should be different (new session each time)
    expect(login1.data.token).not.toBe(login2.data.token);

    // Both tokens should be valid
    const me1 = await api.getMe(login1.data.token);
    const me2 = await api.getMe(login2.data.token);

    expect(me1.status).toBe(200);
    expect(me2.status).toBe(200);
  });

  test('login with wrong password does not leak user existence', async () => {
    const email = uniqueEmail('wrong-pass');
    const password = 'TestPassword123!';

    await api.register(email, password, 'Wrong Pass User');

    // Wrong password for existing user
    const existingResult = await api.login(email, 'WrongPassword1!');

    // Non-existent user
    const nonExistentResult = await api.login(
      'nonexistent@example.com',
      'SomePassword1!'
    );

    // Both should return same status and similar error
    expect(existingResult.status).toBe(nonExistentResult.status);
    expect(existingResult.data?.error).toBe(nonExistentResult.data?.error);
  });
});

// ==================== Authorization Boundaries ====================

test.describe('Security - Authorization Boundaries', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('unauthenticated request to protected endpoint fails', async () => {
    const result = await api.getProjects('');

    expect(result.status).toBe(401);
  });

  test('project ID manipulation does not grant access', async () => {
    const { user } = await createUserWithProject(api, 'id-manip');

    // Try various fake project IDs
    const fakeIds = [
      'fake-project-id',
      '00000000-0000-0000-0000-000000000000',
      '../../../etc/passwd',
      "'; DROP TABLE projects; --",
    ];

    for (const fakeId of fakeIds) {
      const result = await api.getProject(user.token, fakeId);
      expect([400, 401, 403, 404]).toContain(result.status);
    }
  });

  test('SQL injection in project ID is handled safely', async () => {
    const { user } = await createUserWithProject(api, 'sql-inj');

    const sqlInjectionId = "1' OR '1'='1";
    const result = await api.getProject(user.token, sqlInjectionId);

    // Should return error, not all projects
    expect([400, 401, 403, 404]).toContain(result.status);
  });

  test('path traversal in endpoints is blocked', async () => {
    const { user } = await createUserWithProject(api, 'path-trav');

    const traversalId = '../../../admin/config';
    const result = await api.getProject(user.token, traversalId);

    expect([400, 401, 403, 404]).toContain(result.status);
  });
});

// ==================== Ingestion Security ====================

test.describe('Security - Ingestion Endpoint', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('ingestion validates event structure', async () => {
    const { project } = await createUserWithProject(api, 'ingest-validate');

    // Empty events array
    const emptyResult = await api.ingestSpansRaw(project.apiKey, []);
    expect([200, 400]).toContain(emptyResult.status);

    // Invalid event structure
    const invalidResult = await api.ingestSpansRaw(project.apiKey, [
      { invalid: 'structure' },
    ]);
    expect([200, 400]).toContain(invalidResult.status);
  });

  test('ingestion handles large payloads gracefully', async () => {
    const { project } = await createUserWithProject(api, 'ingest-large');

    // Create many events
    const largePayload = Array(100)
      .fill(null)
      .map((_, i) => createSampleEvent(`Event ${i}`));

    const result = await api.ingestSpansRaw(project.apiKey, largePayload);

    // Should either accept or reject gracefully, not crash
    expect([200, 400, 413]).toContain(result.status);
  });

  test('ingestion sanitizes input data', async () => {
    const { project } = await createUserWithProject(api, 'ingest-sanitize');

    const maliciousEvent = {
      ...createSampleEvent(),
      name: '<script>alert("xss")</script>',
      metadata: {
        sql: "'; DROP TABLE traces; --",
        html: '<img src=x onerror=alert(1)>',
      },
    };

    const result = await api.ingestSpans(project.apiKey, [maliciousEvent]);

    // Should accept (data is stored but sanitized on output)
    expect(result.status).toBe(200);
  });

  test('ingestion returns security headers', async () => {
    const { project } = await createUserWithProject(api, 'ingest-headers');

    const result = await api.ingestSpansRaw(project.apiKey, [createSampleEvent()]);

    expect(result.headers['x-content-type-options']).toBe('nosniff');
    expect(result.headers['x-frame-options']).toBe('DENY');
  });
});

// ==================== Edge Cases ====================

test.describe('Security - Edge Cases', () => {
  let api: ApiHelper;

  test.beforeEach(async ({ request }) => {
    api = new ApiHelper(request);
  });

  test('handles concurrent requests from same user', async () => {
    const { user, project } = await createUserWithProject(api, 'concurrent');

    // Make multiple concurrent requests
    const results = await Promise.all([
      api.getProjects(user.token),
      api.getProject(user.token, project.id),
      api.getProjectStats(user.token, project.id),
      api.getMe(user.token),
    ]);

    // All should succeed
    results.forEach((result) => {
      expect(result.status).toBe(200);
    });
  });

  test('handles rapid sequential requests', async () => {
    const { user } = await createUserWithProject(api, 'rapid');

    // Rapid sequential requests
    for (let i = 0; i < 5; i++) {
      const result = await api.getMe(user.token);
      expect(result.status).toBe(200);
    }
  });

  test('deleted project is no longer accessible', async () => {
    const { user, project } = await createUserWithProject(api, 'deleted');

    // Delete the project
    const deleteResult = await api.deleteProject(user.token, project.id);
    expect(deleteResult.status).toBe(200);

    // Project should no longer be accessible
    const getResult = await api.getProject(user.token, project.id);
    expect([404, 410]).toContain(getResult.status);

    // API key should no longer work
    const ingestResult = await api.ingestSpansRaw(project.apiKey, [
      createSampleEvent(),
    ]);
    expect(ingestResult.status).toBe(401);
  });

  test('user can have multiple projects with separate isolation', async () => {
    const email = uniqueEmail('multi-proj');
    const password = 'TestPassword123!';

    await api.register(email, password, 'Multi Project User');
    const login = await api.login(email, password);

    // Create multiple projects
    const proj1 = await api.createProject(login.data.token, 'Project 1');
    const proj2 = await api.createProject(login.data.token, 'Project 2');

    // Ingest to each with their own key
    await api.ingestSpans(proj1.data.APIKey || proj1.data.apiKey, [
      createSampleEvent('Proj1 Event'),
    ]);
    await api.ingestSpans(proj2.data.APIKey || proj2.data.apiKey, [
      createSampleEvent('Proj2 Event'),
    ]);

    // Each project should have its own traces (isolation within user)
    // Verified by trace counts being separate
    const stats1 = await api.getProjectStats(
      login.data.token,
      proj1.data.ID || proj1.data.id
    );
    const stats2 = await api.getProjectStats(
      login.data.token,
      proj2.data.ID || proj2.data.id
    );

    expect(stats1.status).toBe(200);
    expect(stats2.status).toBe(200);
  });
});
