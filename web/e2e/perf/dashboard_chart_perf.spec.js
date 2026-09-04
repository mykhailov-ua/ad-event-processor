import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from '../helpers.js';
import {
  assertPlaywrightBudget,
  medianWallClockMs,
  PLAYWRIGHT_BUDGETS,
} from './helpers_perf.js';

test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('dashboard chart region within perf budget (chart_mock=1)', async ({ page }) => {
  await loginAsAdmin(page);

  const medianMs = await medianWallClockMs(async () => {
    await page.goto('/dashboards/buyer?chart_mock=1&admin_perf=1');
    await expect(page.getByRole('region', { name: 'Key performance indicators' })).toBeVisible();
    await expect(page.getByRole('region', { name: 'Performance chart' })).toBeVisible();
  });

  assertPlaywrightBudget(
    'dashboard chart region visible',
    medianMs,
    PLAYWRIGHT_BUDGETS.chartRegionMs,
  );

  const rowBuildMs = await page.evaluate(() => window.__ADMIN_PERF__?.marks?.['dashboard-chart-rows']);
  if (rowBuildMs != null) {
    assertPlaywrightBudget('dashboard chart row build', rowBuildMs, 50);
  }
});
