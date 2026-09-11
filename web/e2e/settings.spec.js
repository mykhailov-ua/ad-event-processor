import { test, expect } from '@playwright/test';

import {
  gotoLive,
  isApiGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings page loads platform settings from GET /api/v1/settings/platform', async ({ page }) => {
  await loginAsAdmin(page);
  const settingsResponse = page.waitForResponse(isApiGet('/api/v1/settings/platform'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/settings');
  await expect(mainHeading(page, 'Settings')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply license' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save changes' })).toBeVisible();

  const response = await settingsResponse;
  const body = await response.json();
  expect(body).toHaveProperty('config');
  expect(body).toHaveProperty('bootstrap_complete');
  if (body.bootstrap_complete) {
    await expect(page.getByRole('button', { name: 'Write install.compose.env' })).toBeEnabled();
  } else {
    await expect(page.getByRole('button', { name: 'Write install.compose.env' })).toBeDisabled();
  }
});
