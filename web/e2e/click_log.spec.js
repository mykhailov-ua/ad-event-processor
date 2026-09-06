import { test, expect } from '@playwright/test';

import { gotoLiveAwaitGet, isApiGet, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('click log page loads from GET /api/v1/reports/click-log', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/reports/click-log?admin_dev=0');
  await expect(page.getByRole('heading', { name: /click log/i })).toBeVisible();

  const clickLogGet = page.waitForResponse(isApiGet('/api/v1/reports/click-log'), {
    timeout: 30_000,
  });
  await page.getByRole('button', { name: /apply/i }).click();
  const response = await clickLogGet;
  expect(response.ok()).toBe(true);
  const body = await response.json();
  expect(body).toHaveProperty('items');
  expect(Array.isArray(body.items)).toBe(true);

  if (body.items.length === 0) {
    await expect(page.getByText(/no click events/i).first()).toBeVisible({ timeout: 15_000 });
    return;
  }
  await expect(page.getByText(/click id/i).first()).toBeVisible({ timeout: 15_000 });
});
