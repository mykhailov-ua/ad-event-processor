import { test, expect } from '@playwright/test';

import {
  gotoLive,
  gotoLiveAwaitGet,
  isApiGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

const FAILED_EXPORT_JOB_ID = '00000000-0000-4000-8000-00000000e801';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'export hub loads catalog from GET /api/v1/reports/catalog',
  { tag: '@L1' },
  async ({ page }) => {
    await loginAsAdmin(page);
    const catalogResponse = await gotoLiveAwaitGet(page, '/exports', '/api/v1/reports/catalog');
    expect(catalogResponse.ok()).toBe(true);

    await expect(mainHeading(page, 'Exports')).toBeVisible();
    await expect(page.getByLabel('Export')).toBeVisible();
  }
);

test('export hub failed job poll shows ErrorBlock', { tag: '@L3' }, async ({ page }) => {
  await loginAsAdmin(page);
  const catalogResponse = page.waitForResponse(isApiGet('/api/v1/reports/catalog'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/exports');
  await catalogResponse;

  await page.route(`**/api/v1/reports/jobs/${FAILED_EXPORT_JOB_ID}`, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: FAILED_EXPORT_JOB_ID,
        status: 'failed',
        error: 'clickhouse query timeout',
      }),
    });
  });

  await page.locator('#export-hub-job-id').fill(FAILED_EXPORT_JOB_ID);
  const jobGet = page.waitForResponse(
    (response) =>
      response.request().method() === 'GET' &&
      response.url().includes(`/api/v1/reports/jobs/${FAILED_EXPORT_JOB_ID}`),
    { timeout: 20_000 }
  );
  await page.getByTestId('export-job-refresh').click();
  await jobGet;

  await expect(page.getByTestId('export-job-error')).toBeVisible();
  await expect(page.getByText('Export failed', { exact: true })).toBeVisible();
  await expect(page.getByText('clickhouse query timeout', { exact: true })).toBeVisible();
});
