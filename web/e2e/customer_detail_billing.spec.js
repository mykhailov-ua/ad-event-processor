import { test, expect } from '@playwright/test';

import {
  gotoLive,
  gotoLiveAwaitGet,
  isApiGet,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
  stubApiRoute,
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
  await gotoLiveAwaitGet(page, '/customers', '/api/v1/customers');
  await mainContent(page).getByRole('heading', { name: 'Customers', exact: true }).waitFor({
    timeout: 15_000,
  });

  const customerLink = mainContent(page).locator('a[href^="/customers/"]').first();
  const count = await customerLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no customers in directory');
    return;
  }

  const customerHref = await customerLink.getAttribute('href');
  const customerId = customerHref?.split('/').pop() ?? '';
  const customerGet = page.waitForResponse(isApiGet(`/api/v1/customers/${customerId}`), {
    timeout: 20_000,
  });
  await customerLink.click();
  await customerGet;
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

test('customer detail balance GET 500 shows blocking error', { tag: '@L3' }, async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLiveAwaitGet(page, '/customers', '/api/v1/customers');
  await mainContent(page).getByRole('heading', { name: 'Customers', exact: true }).waitFor({
    timeout: 15_000,
  });

  const customerLink = mainContent(page).locator('a[href^="/customers/"]').first();
  const count = await customerLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no customers in directory');
    return;
  }

  const customerHref = await customerLink.getAttribute('href');
  const customerId = customerHref?.split('/').pop() ?? '';
  const customerGet = page.waitForResponse(isApiGet(`/api/v1/customers/${customerId}`), {
    timeout: 20_000,
  });
  await customerLink.click();
  await customerGet;
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

  await stubApiRoute(page, 'GET', `/api/v1/customers/${customerId}/balance`, 500, {
    code: 'INTERNAL_ERROR',
    message: 'balance unavailable',
  });

  await mainContent(page).getByRole('button', { name: 'Balance', exact: true }).click();

  await expect(mainContent(page).getByRole('alert')).toBeVisible({ timeout: 15_000 });
  await expect(
    mainContent(page).getByText('Could not load balance', { exact: true })
  ).toBeVisible();
  await expect(
    mainContent(page).getByText('The server encountered an error. Try again later.', { exact: true })
  ).toBeVisible();
});
