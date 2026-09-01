import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('rtb page loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/rtb');
  await expect(page.getByRole('heading', { name: 'RTB' })).toBeVisible();
});
