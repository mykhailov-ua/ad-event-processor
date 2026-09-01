import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud decision page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/decisions');
  await expect(page.getByRole('heading', { name: 'Fraud decision explain' })).toBeVisible();
});
