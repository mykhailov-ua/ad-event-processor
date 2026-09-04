import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('rtb deals list loads from GET /api/v1/rtb/deals when openrtb licensed', async ({ page }, testInfo) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/rtb/deals'), { timeout: 20_000 });
  await gotoLive(page, '/rtb/deals');

  const licenseStub = page.getByText('OpenRTB license required', { exact: true });
  if (await licenseStub.isVisible({ timeout: 5000 }).catch(() => false)) {
    testInfo.skip(true, 'integration: openrtb license not enabled on stack');
    return;
  }

  await expect(page.getByRole('heading', { name: 'RTB deals' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'RTB sections' })).toBeVisible();

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No deals',
    rowLabel: (row) => String(row.name ?? row.id ?? ''),
  });
});
