import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud hub links are visible', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud');
  await expect(page.getByRole('heading', { name: 'Fraud' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Integrations' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'ML labels' })).toBeVisible();
});
