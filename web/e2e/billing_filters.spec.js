import { test, expect } from '@playwright/test';

import { gotoBilling, isApiGet, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('billing invoice filters are visible', async ({ page }) => {
  await loginAsAdmin(page);
  const invoicesGet = page.waitForResponse(isApiGet('/api/v1/billing/invoices'), {
    timeout: 20_000,
  });
  await gotoBilling(page);
  await invoicesGet;

  await expect(page.getByRole('heading', { name: 'Invoices' })).toBeVisible();
  await expect(page.getByLabel('Month')).toBeVisible();
  await expect(page.getByLabel('Status')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply' })).toBeVisible();
});

test('billing apply filter updates query string and refetches invoices', async ({ page }) => {
  await loginAsAdmin(page);
  const initialGet = page.waitForResponse(isApiGet('/api/v1/billing/invoices'), {
    timeout: 20_000,
  });
  await gotoBilling(page);
  await initialGet;

  await page.getByLabel('Month').fill('2026-01');
  const filteredGet = page.waitForResponse(
    (response) =>
      isApiGet('/api/v1/billing/invoices')(response) && response.url().includes('month=2026-01'),
    { timeout: 20_000 }
  );
  await page.getByRole('button', { name: 'Apply' }).click();
  const response = await filteredGet;
  expect(response.ok()).toBe(true);

  await expect(page).toHaveURL(/month=2026-01/);
});
