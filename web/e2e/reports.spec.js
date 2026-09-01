import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('reports catalog renders cards', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/reports');
  await expect(page.getByRole('heading', { name: 'Reports' })).toBeVisible();
});

test('report runner page loads from catalog link', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/reports');
  const firstLink = page.locator('main a[href^="/reports/"]').first();
  const count = await firstLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no report catalog rows for session');
    return;
  }
  await firstLink.click();
  await expect(page.getByRole('button', { name: 'Run report' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Back to catalog' })).toBeVisible();
});
