import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings bootstrap section when first-run install is pending', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/settings');

  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();

  const bootstrapHeading = page.getByRole('heading', { name: 'Initial setup' });
  if (await bootstrapHeading.isVisible()) {
    await expect(page.getByLabel('Setup token')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Complete setup' })).toBeVisible();
  }
});
