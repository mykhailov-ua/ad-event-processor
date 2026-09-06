import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from '../helpers.js';
import { assertPlaywrightBudget, medianWallClockMs, PLAYWRIGHT_BUDGETS } from './helpers_perf.js';

test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('command palette open within perf budget', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/campaigns');

  const medianMs = await medianWallClockMs(async () => {
    await page.keyboard.press('Control+k');
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByLabel('Command palette search')).toBeVisible();
  });

  assertPlaywrightBudget('command palette open', medianMs, PLAYWRIGHT_BUDGETS.paletteOpenMs);
});
