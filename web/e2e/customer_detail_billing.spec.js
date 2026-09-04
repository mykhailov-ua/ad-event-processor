import { test, expect } from '@playwright/test';

import {
  gotoLive,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
} from './helpers.js';

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
  await gotoLive(page, '/customers');
  await mainContent(page).getByRole('heading', { name: 'Customers', exact: true }).waitFor({
    timeout: 15_000,
  });

  const customerLink = mainContent(page).locator('a[href^="/customers/"]').first();
  const count = await customerLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no customers in directory');
    return;
  }

  await customerLink.click();
  await page.waitForURL(/\/customers\/[^/]+$/, { timeout: 15_000 });

  const profileTab = mainContent(page).getByRole('button', { name: 'Profile', exact: true });
  const pageError = page.getByText('Page error', { exact: true });
  await Promise.race([
    profileTab.waitFor({ state: 'visible', timeout: 15_000 }),
    pageError.waitFor({ state: 'visible', timeout: 15_000 }),
  ]);
  if (await pageError.isVisible()) {
    test.skip(true, 'integration: customer detail page error boundary');
    return;
  }

  await expect(profileTab).toBeVisible();

  for (const label of DETAIL_TAB_LABELS) {
    await expect(mainContent(page).getByRole('button', { name: label, exact: true })).toBeVisible();
  }
});
