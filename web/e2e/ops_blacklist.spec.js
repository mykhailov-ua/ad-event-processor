import { test, expect } from '@playwright/test';

import {
  gotoLiveAwaitGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('ops blacklist page loads from GET /api/v1/ops/blacklist', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = await gotoLiveAwaitGet(page, '/ops/blacklist', '/api/v1/ops/blacklist');
  await expect(mainHeading(page, 'Fraud blacklist')).toBeVisible();
  expect(listResponse.ok()).toBe(true);
  const body = await listResponse.json();
  expect(body).toBeTruthy();
});
