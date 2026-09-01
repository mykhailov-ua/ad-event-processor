import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('rtb deals list loads when openrtb licensed', async ({ page }, testInfo) => {
  await loginAsAdmin(page);
  await page.goto('/rtb/deals');

  const licenseStub = page.getByText('OpenRTB license required', { exact: true });
  if (await licenseStub.isVisible({ timeout: 5000 }).catch(() => false)) {
    testInfo.skip(true, 'integration: openrtb license not enabled on stack');
    return;
  }

  await expect(page.getByRole('heading', { name: 'RTB deals' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'RTB sections' })).toBeVisible();

  const table = page.getByRole('table');
  const empty = page.getByText('No deals', { exact: true });
  await expect(table.or(empty)).toBeVisible({ timeout: 15_000 });
});
