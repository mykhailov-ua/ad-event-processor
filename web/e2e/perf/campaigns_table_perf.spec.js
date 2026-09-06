import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from '../helpers.js';
import { assertPlaywrightBudget, medianWallClockMs, PLAYWRIGHT_BUDGETS } from './helpers_perf.js';

test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaigns table first paint within perf budget', async ({ page }) => {
  await loginAsAdmin(page);

  const medianMs = await medianWallClockMs(async () => {
    await page.goto('/campaigns');
    await expect(page.getByRole('heading', { name: /^campaigns$/i })).toBeVisible();
    const table = page.getByRole('table');
    await expect(table).toBeVisible();
    await expect(table.getByRole('row').nth(1)).toBeVisible();
  });

  assertPlaywrightBudget(
    'campaigns table first paint',
    medianMs,
    PLAYWRIGHT_BUDGETS.campaignTableMs
  );
});
