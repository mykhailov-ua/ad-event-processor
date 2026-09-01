import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud integrations page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/integrations');
  await expect(page.getByRole('heading', { name: 'Fraud integrations' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Load' })).toBeVisible();
});
