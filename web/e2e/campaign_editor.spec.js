import { test, expect } from '@playwright/test';

import {
  gotoCampaignsLive,
  isApiGet,
  isCampaignsListResponse,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign editor opens from directory and loads GET campaign', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });
  await gotoCampaignsLive(page);
  const listBody = await (await listResponse).json();

  const firstCampaign = listBody.items?.[0];
  if (!firstCampaign?.id) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  const campaignGet = page.waitForResponse(isApiGet(`/api/v1/campaigns/${firstCampaign.id}`), {
    timeout: 20_000,
  });
  await page.locator('main a[href$="/edit"]').first().click();
  const response = await campaignGet;
  expect(response.ok()).toBe(true);

  await expect(page.getByRole('heading', { name: 'Campaign settings' })).toBeVisible();
  await expect(page.getByLabel('Name')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save' })).toBeVisible();
});
