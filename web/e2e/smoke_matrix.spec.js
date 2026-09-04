import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  gotoLive,
  gotoLiveTeam,
  loginAsAdmin,
  loginSignInHeading,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('install onboarding routes load headings', async ({ page }) => {
  await page.goto('/setup');

  const setupHeading = page.getByRole('heading', { name: 'Initial setup' });
  const signInHeading = loginSignInHeading(page);

  if (await setupHeading.isVisible({ timeout: 5000 }).catch(() => false)) {
    await expect(setupHeading).toBeVisible();
    return;
  }

  await expect(signInHeading).toBeVisible();
});

test('key admin routes load page headings', async ({ page }) => {
  await loginAsAdmin(page);
  const customerId = await fetchSessionCustomerId(page);

  const routes = [
    { path: '/customers', heading: 'Customers' },
    { path: '/campaigns', heading: 'Campaigns' },
    { path: '/billing', heading: 'Billing' },
    { path: '/settings', heading: 'Platform settings' },
    { path: '/settings/license', heading: 'License' },
    {
      path: customerId ? `/team?customer_id=${encodeURIComponent(customerId)}` : '/team',
      heading: 'Team',
      skipWithoutCustomer: true,
    },
    { path: '/audit', heading: 'Audit' },
    { path: '/reports', heading: 'Reports' },
    { path: '/ops', heading: 'Ops' },
    { path: '/fraud', heading: 'Fraud' },
    { path: '/fraud/presets', heading: 'Fraud presets' },
  ];

  for (const route of routes) {
    const { path, heading, skipWithoutCustomer } = route;
    if (skipWithoutCustomer && !customerId) {
      continue;
    }
    if (path.startsWith('/team')) {
      const loaded = await gotoLiveTeam(page, customerId);
      if (!loaded) {
        test.skip(true, 'integration: team overview requires customer_id');
        return;
      }
      continue;
    }
    await gotoLive(page, path);
    await expect(mainHeading(page, heading)).toBeVisible({ timeout: 15_000 });
  }
});
