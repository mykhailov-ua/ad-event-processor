import { test, expect } from '@playwright/test';

import { isApiGet, loginAsAdmin, mainNav, skipUnlessIntegrationReady } from './helpers.js';

const NAV_LABELS = ['Dashboard', 'Campaigns', 'Users', 'Billing', 'Maintenance'];

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('sidebar nav links are visible after login and session loads', async ({ page }) => {
  const sessionGet = page.waitForResponse(isApiGet('/api/v1/session'), { timeout: 20_000 });
  await loginAsAdmin(page);
  const sessionResponse = await sessionGet;
  expect(sessionResponse.ok()).toBe(true);

  const nav = mainNav(page);
  for (const label of NAV_LABELS) {
    await expect(nav.getByRole('link', { name: label })).toBeVisible();
  }
});
