import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/settings');
  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
});
