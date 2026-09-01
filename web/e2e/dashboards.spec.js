import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('dashboard buyer shows chart and breakdown sections', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboards/buyer');
  await expect(page.getByRole('heading', { name: /^dashboard$/i })).toBeVisible();
  await page.getByRole('button', { name: /^apply$/i }).click();
  await expect(page.locator('.recharts-responsive-container').first()).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText('Campaigns', { exact: true })).toBeVisible();
  await expect(page.getByText('Landing pages', { exact: true })).toBeVisible();
  await expect(page.getByText('Recent clicks', { exact: true })).toBeVisible();
});
