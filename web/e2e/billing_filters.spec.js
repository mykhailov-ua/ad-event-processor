import { test, expect } from '@playwright/test';

import { gotoBilling, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('billing invoice filters are visible', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  await expect(page.getByRole('heading', { name: 'Invoices' })).toBeVisible();
  await expect(page.getByLabel('Month')).toBeVisible();
  await expect(page.getByLabel('Status')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply' })).toBeVisible();
});

test('billing apply filter updates query string', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  await page.getByLabel('Month').fill('2026-01');
  await page.getByRole('button', { name: 'Apply' }).click();

  await expect(page).toHaveURL(/month=2026-01/);
});
