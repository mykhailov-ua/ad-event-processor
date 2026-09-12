import { test, expect } from '@playwright/test';

import {
  gotoCampaignsLive,
  isCampaignsListResponse,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'bulk edit dialog posts bulk-patch and refreshes campaign list',
  { tag: '@write' },
  async ({ page }) => {
    await loginAsAdmin(page);
    const initialList = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });
    await gotoCampaignsLive(page);
    const listResponse = await initialList;
    const listBody = await listResponse.json();

    const activeCampaign = listBody.items.find((row) => row.status === 'ACTIVE');
    if (!activeCampaign) {
      test.skip(true, 'integration: no ACTIVE campaign for bulk patch');
      return;
    }

    const rowCheckbox = page.getByRole('checkbox', {
      name: `Select ${activeCampaign.name}`,
      exact: true,
    });
    await expect(rowCheckbox).toBeVisible({ timeout: 15_000 });
    await rowCheckbox.check();

    await page.getByRole('button', { name: /Bulk edit/ }).click();
    await expect(page.getByRole('dialog', { name: 'Bulk edit campaigns' })).toBeVisible({
      timeout: 10_000,
    });

    const bulkPatchPost = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        response.url().includes('/api/v1/campaigns/bulk-patch') &&
        response.status() === 200,
      { timeout: 20_000 }
    );
    const refreshedList = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });

    await page.locator('#bulk-patch-timezone').check();
    await page.locator('#bulk-patch-timezone-input').fill('UTC');
    await page.getByRole('button', { name: /Apply to \d+ campaign/ }).click();

    const bulkResponse = await bulkPatchPost;
    const bulkBody = await bulkResponse.json();
    expect(bulkBody.results).toEqual(
      expect.arrayContaining([expect.objectContaining({ id: activeCampaign.id, ok: true })])
    );

    await refreshedList;
  }
);
