import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('ops dlq inbox page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/ops/dlq');
  await expect(page.getByRole('heading', { name: 'DLQ inbox' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Ops sections' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Home' })).toBeVisible();
});
