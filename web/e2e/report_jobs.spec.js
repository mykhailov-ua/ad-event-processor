import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('report jobs page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/reports/jobs');
  await expect(page.getByRole('heading', { name: 'Report export jobs' })).toBeVisible();
});
