import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud presets page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/presets');
  await expect(page.getByRole('heading', { name: 'Fraud presets' })).toBeVisible();
});
