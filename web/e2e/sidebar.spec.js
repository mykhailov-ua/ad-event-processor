import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

const NAV_LABELS = ['Customers', 'Campaigns', 'Billing', 'Ops'];

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('sidebar nav links are visible after login', async ({ page }) => {
  await loginAsAdmin(page);

  const nav = page.getByRole('navigation');
  for (const label of NAV_LABELS) {
    await expect(nav.getByRole('link', { name: label })).toBeVisible();
  }
});
