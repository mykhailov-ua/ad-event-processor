import { test, expect } from '@playwright/test';

import {
  expectErrorBlockVisible,
  gotoLive,
  gotoLiveAwaitGet,
  isApiGet,
  isApiPatch,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
  stubApiRoute,
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

test('settings shows ErrorBlock when GET /api/v1/meta returns 500', { tag: '@L3' }, async ({
  page,
}) => {
  await loginAsAdmin(page);
  await stubApiRoute(page, '/api/v1/meta', 500, {
    error: { code: 'INTERNAL_ERROR', message: 'meta unavailable' },
  });
  const metaResponse = page.waitForResponse(isApiGet('/api/v1/meta', 500), { timeout: 20_000 });
  await gotoLive(page, '/settings');
  await metaResponse;
  await expect(mainHeading(page, 'Settings')).toBeVisible();
  await expectErrorBlockVisible(page, 'Could not load deployment metadata');
});

test('settings PATCH 409 shows platform save error', { tag: '@L3' }, async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLiveAwaitGet(page, '/settings', '/api/v1/settings/platform');

  await stubApiRoute(page, 'PATCH', '/api/v1/settings/platform', 409, {
    message: 'settings revision conflict',
  });

  await page.locator('#settings-tracking-domain').fill('e2e-conflict.example.com');

  const patchResponse = page.waitForResponse(isApiPatch('/api/v1/settings/platform', 409), {
    timeout: 20_000,
  });
  await page.getByRole('button', { name: 'Save changes' }).click();
  await patchResponse;

  await expectErrorBlockVisible(page, 'Could not save platform settings');
  await expect(page.getByText('settings revision conflict', { exact: true })).toBeVisible();
});
