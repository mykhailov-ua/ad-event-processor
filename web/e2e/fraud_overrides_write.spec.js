import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud overrides write form is visible without submitting', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/overrides');
  await expect(page.getByRole('heading', { name: 'Fraud overrides' })).toBeVisible();
  await expect(page.getByLabel('Provide IP hash or raw IP')).toBeVisible();
  await expect(page.getByLabel('IP address')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply override' })).toBeVisible();
});
