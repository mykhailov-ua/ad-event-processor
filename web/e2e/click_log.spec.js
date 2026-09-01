import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('click log page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/reports/click-log');
  await expect(page.getByRole('heading', { name: /click log/i })).toBeVisible();
  await page.getByRole('button', { name: /apply/i }).click();
  await expect(page.getByText(/no click events|click id/i).first()).toBeVisible({
    timeout: 30_000,
  });
});
