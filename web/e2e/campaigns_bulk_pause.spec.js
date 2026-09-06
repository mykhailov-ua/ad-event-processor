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
  'toolbar pause posts bulk action and refreshes campaign list',
  { tag: '@write' },
  async ({ page }) => {
    await loginAsAdmin(page);
    const initialList = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });
    await gotoCampaignsLive(page);
    const listResponse = await initialList;
    const listBody = await listResponse.json();

    const activeCampaign = listBody.items.find((row) => row.status === 'ACTIVE');
    if (!activeCampaign) {
      test.skip(true, 'integration: no ACTIVE campaign for bulk pause');
      return;
    }

    const rowCheckbox = page.getByRole('checkbox', {
      name: `Select ${activeCampaign.name}`,
      exact: true,
    });
    await expect(rowCheckbox).toBeVisible({ timeout: 15_000 });
    await rowCheckbox.check();

    const bulkPost = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        response.url().includes('/api/v1/campaigns/bulk') &&
        response.status() === 200,
      { timeout: 20_000 }
    );
    const refreshedList = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });

    await page.getByRole('button', { name: 'Pause', exact: true }).click();

    const bulkResponse = await bulkPost;
    const bulkBody = await bulkResponse.json();
    expect(bulkBody.results).toEqual(
      expect.arrayContaining([expect.objectContaining({ id: activeCampaign.id, ok: true })])
    );

    const refreshResponse = await refreshedList;
    const refreshBody = await refreshResponse.json();
    const updated = refreshBody.items.find((row) => row.id === activeCampaign.id);
    expect(updated?.status).toBe('PAUSED');
  }
);
