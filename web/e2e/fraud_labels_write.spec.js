import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud labels write form is visible without submitting', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/labels');
  await expect(page.getByRole('heading', { name: 'Fraud labels' })).toBeVisible();
  await expect(page.getByLabel('IP hash (32 hex)')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Upsert label' })).toBeVisible();
});
