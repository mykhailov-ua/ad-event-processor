import { test, expect } from '@playwright/test';

import {
  ensureCustomerScopeLoaded,
  fetchSessionCustomerId,
  gotoLive,
  integrationRunToken,
  integrationSettingsTrackingDomain,
  isApiGet,
  isApiPatch,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings patch merges platform configuration', async ({ page }) => {
  await loginAsAdmin(page);
  const runToken = integrationRunToken();
  const trackingDomain = integrationSettingsTrackingDomain(runToken);

  const initialSettings = page.waitForResponse(isApiGet('/api/v1/settings/platform'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/settings');
  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  await initialSettings;

  const patchBody = JSON.stringify({
    config: { tracking_domain: trackingDomain },
  });
  await page.getByLabel('Configuration changes').fill(patchBody);

  const patchResponse = page.waitForResponse(isApiPatch('/api/v1/settings/platform', 200), {
    timeout: 20_000,
  });
  const refreshedSettings = page.waitForResponse(isApiGet('/api/v1/settings/platform'), {
    timeout: 20_000,
  });

  await page.getByRole('button', { name: 'Apply changes' }).click();

  const patchResult = await patchResponse;
  const patchJson = await patchResult.json();
  expect(patchJson).toBeTruthy();

  await refreshedSettings;

  await expect(page.getByRole('status')).toContainText('Configuration updated.', {
    timeout: 15_000,
  });
});
