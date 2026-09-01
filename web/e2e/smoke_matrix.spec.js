import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('install onboarding routes load headings', async ({ page }) => {
  await page.goto('/setup');

  const setupHeading = page.getByRole('heading', { name: 'Initial setup' });
  const signInHeading = page.getByRole('heading', { name: 'Sign in' });

  if (await setupHeading.isVisible({ timeout: 5000 }).catch(() => false)) {
    await expect(setupHeading).toBeVisible();
    return;
  }

  await expect(signInHeading).toBeVisible();
});

test('key admin routes load page headings', async ({ page }) => {
  await loginAsAdmin(page);

  const routes = [
    { path: '/customers', heading: 'Customers' },
    { path: '/campaigns', heading: 'Campaigns' },
    { path: '/billing', heading: 'Billing' },
    { path: '/settings', heading: 'Platform settings' },
    { path: '/settings/license', heading: 'License' },
    { path: '/team', heading: 'Team' },
    { path: '/audit', heading: 'Audit' },
    { path: '/reports', heading: 'Reports' },
    { path: '/ops', heading: 'Ops' },
    { path: '/fraud', heading: 'Fraud' },
    { path: '/fraud/presets', heading: 'Fraud presets' },
    { path: '/rtb', heading: 'RTB' },
  ];

  for (const { path, heading } of routes) {
    await page.goto(path);
    await expect(page.getByRole('heading', { name: heading })).toBeVisible();
  }
});
