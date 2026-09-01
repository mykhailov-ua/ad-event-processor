import { test, expect } from '@playwright/test';

import { gotoBilling, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('billing page shows ledger exports and operator tools', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  await expect(page.getByRole('link', { name: 'Ledger exports' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Ledger invariant' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Invoice preview' })).toBeVisible();
});

test('billing grid opens invoice detail when invoices exist', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  const invoiceLink = page.locator('main a[href^="/billing/invoices/"]').first();
  const count = await invoiceLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no invoices in billing grid');
    return;
  }

  await invoiceLink.click();
  await expect(page.getByRole('button', { name: 'Download PDF' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Ledger lines' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Deliveries' })).toBeVisible();
});

test('billing exports page loads from billing hub', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  await page.getByRole('link', { name: 'Ledger exports' }).click();
  await expect(page.getByRole('heading', { name: 'Billing ledger exports' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Start export' })).toBeVisible();
});
