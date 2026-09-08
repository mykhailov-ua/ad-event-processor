import { test, expect } from '@playwright/test';

import {
  gotoLiveAwaitGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('reports catalog renders cards from GET /api/v1/reports/catalog', async ({ page }) => {
  await loginAsAdmin(page);
  const catalogResponse = await gotoLiveAwaitGet(page, '/reports', '/api/v1/reports/catalog');
  await expect(mainHeading(page, 'Reports')).toBeVisible();
  const body = await catalogResponse.json();
  expect(body).toHaveProperty('rows');
  expect(Array.isArray(body.rows)).toBe(true);
});

test('report runner page loads from catalog link', async ({ page }) => {
  await loginAsAdmin(page);
  const catalogResponse = await gotoLiveAwaitGet(page, '/reports', '/api/v1/reports/catalog');
  const catalogBody = await catalogResponse.json();
  const reports = catalogBody.rows ?? [];
  if (reports.length === 0) {
    test.skip(true, 'integration: no report catalog rows for session');
    return;
  }

  const firstKey = String(reports[0].key ?? '');
  const runnerHref = firstKey.includes('/') ? `/reports/${encodeURIComponent(firstKey)}` : `/reports/${firstKey}`;
  await page.locator(`main a[href="${runnerHref}"]`).first().click();
  await expect(page.getByRole('button', { name: 'Run report' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Back to catalog' })).toBeVisible();
});
