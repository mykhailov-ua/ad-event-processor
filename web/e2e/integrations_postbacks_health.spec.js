import { test, expect } from '@playwright/test';

import { gotoLive, isApiGet, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('postbacks health tab loads from GET /api/v1/postbacks/health', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/integrations/postbacks');

  const healthGet = page.waitForResponse(isApiGet('/api/v1/postbacks/health'), { timeout: 20_000 });
  await page.getByRole('button', { name: 'Health' }).click();

  const response = await healthGet;
  expect(response.status()).toBe(200);
  const body = await response.json();
  expect(Array.isArray(body.rows)).toBe(true);
  expect(body.alert_threshold_success_rate).toBe(95);

  await expect(page.getByRole('heading', { name: 'Delivery health (24h)' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Runbook' })).toBeVisible();
});
