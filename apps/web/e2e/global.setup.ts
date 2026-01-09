import { test as setup, expect } from '@playwright/test';
import { ApiHelper } from './helpers/api';
import { testUsers } from './fixtures/test-data';

/**
 * Global setup - runs once before all tests.
 * Creates test users and verifies backend is running.
 */
setup('global setup', async ({ request }) => {
  const api = new ApiHelper(request);

  // 1. Check backend is running
  console.log('Checking backend health...');
  const isHealthy = await api.healthCheck();
  if (!isHealthy) {
    console.warn('⚠️  Backend is not running. Some tests may fail.');
    console.warn('    Start backend with: cd apps/server && go run ./cmd/server');
  } else {
    console.log('✓ Backend is healthy');
  }

  // 2. Check features endpoint
  console.log('Checking features endpoint...');
  try {
    const features = await api.getFeatures();
    console.log(`✓ Edition: ${features.edition}`);
    console.log(`  Features: ${JSON.stringify(features.features)}`);

    // Store for use in tests
    process.env.LELEMON_EDITION = features.edition;
    process.env.LELEMON_FEATURES = JSON.stringify(features.features);
  } catch (error) {
    console.warn('⚠️  Could not fetch features');
  }

  // 3. Create test user (if backend is running)
  if (isHealthy) {
    console.log('Creating test user...');
    const result = await api.register(
      testUsers.default.email,
      testUsers.default.password,
      testUsers.default.name
    );

    if (result.status === 201) {
      console.log('✓ Test user created');
    } else if (result.status === 409) {
      console.log('✓ Test user already exists');
    } else {
      console.warn(`⚠️  Could not create test user: ${result.status}`);
    }
  }

  console.log('Setup complete!');
});
