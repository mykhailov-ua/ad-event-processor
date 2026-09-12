import { test, expect } from '@playwright/test';

import {
  expectErrorBlockVisible,
  gotoCampaignsLive,
  isApiGet,
  isApiPatch,
  isCampaignsListResponse,
  loginAsAdmin,
  openFirstCampaignEditor,
  skipUnlessIntegrationReady,
  stubApiRoute,
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

test('campaign editor PATCH 400 shows save error with field message', { tag: '@L3' }, async ({
  page,
}) => {
  await loginAsAdmin(page);
  const campaignId = await openFirstCampaignEditor(page);
  if (!campaignId) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await page.getByLabel('Name').fill('E2E invalid rename');

  await stubApiRoute(page, 'PATCH', `/api/v1/campaigns/${campaignId}`, 400, {
    message: 'name: invalid campaign name',
  });

  const patchResponse = page.waitForResponse(isApiPatch(`/api/v1/campaigns/${campaignId}`, 400), {
    timeout: 20_000,
  });
  await page.getByRole('button', { name: 'Save' }).click();
  await patchResponse;

  await expectErrorBlockVisible(page, 'Could not save campaign');
  await expect(page.getByText('name: invalid campaign name', { exact: true })).toBeVisible();
});
