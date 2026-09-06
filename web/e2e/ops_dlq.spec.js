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

test('ops dlq inbox page loads from GET /api/v1/ops/dlq/inbox', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = await gotoLiveAwaitGet(page, '/ops/dlq', '/api/v1/ops/dlq/inbox');
  await expect(mainHeading(page, 'DLQ inbox')).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Ops sections' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Home' })).toBeVisible();
  expect(listResponse.ok()).toBe(true);
  const body = await listResponse.json();
  expect(body).toBeTruthy();
});
