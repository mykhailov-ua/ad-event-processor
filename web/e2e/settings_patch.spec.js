import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings patch form is visible', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/settings');

  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  await expect(page.getByLabel('Configuration changes')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply changes' })).toBeVisible();
});
