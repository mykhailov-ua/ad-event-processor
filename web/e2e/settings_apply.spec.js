import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings apply to disk form is visible', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/settings');

  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Save configuration to disk' })).toBeVisible();
  await expect(page.getByLabel('Installation directory (optional)')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save to disk' })).toBeVisible();
});
