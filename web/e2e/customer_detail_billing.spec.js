import { test, expect } from '@playwright/test';

import { gotoCustomers, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

const DETAIL_TAB_LABELS = [
  'Profile',
  'Balance',
  'Ledger',
  'Statement',
  'Forecast',
  'Wallet',
  'Payments',
  'Tax profile',
];

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('customer detail shows billing tab bar', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCustomers(page);

  const customerLink = page.locator('main a[href^="/customers/"]').first();
  const count = await customerLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no customers in directory');
    return;
  }

  await customerLink.click();

  for (const label of DETAIL_TAB_LABELS) {
    await expect(page.getByRole('button', { name: label, exact: true })).toBeVisible();
  }
});
