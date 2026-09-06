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

test('settings page loads platform config from GET /api/v1/settings/platform', async ({ page }) => {
  await loginAsAdmin(page);
  const settingsResponse = await gotoLiveAwaitGet(page, '/settings', '/api/v1/settings/platform');
  await expect(mainHeading(page, 'Platform settings')).toBeVisible();
  expect(settingsResponse.ok()).toBe(true);
  const body = await settingsResponse.json();
  expect(body).toBeTruthy();
});
