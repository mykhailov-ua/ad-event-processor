import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings license route loads for authenticated operator', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/settings/license');

  await expect(page.getByRole('heading', { name: 'License' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply license' })).toBeVisible();
});
