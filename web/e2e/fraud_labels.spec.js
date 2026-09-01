import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud labels page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/labels');
  await expect(page.getByRole('heading', { name: 'Fraud labels' })).toBeVisible();
});
