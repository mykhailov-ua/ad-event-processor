import { test, expect } from '@playwright/test';

import { gotoLive, isApiGet, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('rtb overview loads from GET /api/v1/reports/rtb/overview when openrtb licensed', async ({
  page,
}, testInfo) => {
  await loginAsAdmin(page);
  const overviewGet = page.waitForResponse(isApiGet('/api/v1/reports/rtb/overview'), {
    timeout: 30_000,
  });
  await gotoLive(page, '/rtb');

  const licenseStub = page.getByText('OpenRTB license required', { exact: true });
  if (await licenseStub.isVisible({ timeout: 5000 }).catch(() => false)) {
    testInfo.skip(true, 'integration: openrtb license not enabled on stack');
    return;
  }

  await expect(page.getByRole('heading', { name: 'RTB' })).toBeVisible();

  const response = await overviewGet;
  expect(response.ok()).toBe(true);
  const body = await response.json();
  expect(body).toHaveProperty('rows');
  expect(Array.isArray(body.rows)).toBe(true);

  await expect(page.getByRole('navigation', { name: 'RTB sections' })).toBeVisible({
    timeout: 15_000,
  });
});
