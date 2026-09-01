import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('team page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/team');
  await expect(page.getByRole('heading', { name: 'Team' })).toBeVisible();
});
