import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('ops blacklist page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/ops/blacklist');
  await expect(page.getByRole('heading', { name: 'Fraud blacklist' })).toBeVisible();
});
