import { test, expect } from '@playwright/test';

import {
  fetchFirstCampaignId,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
  stubApiRoute,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('postbacks config PUT 400 shows save error', { tag: '@L3' }, async ({ page }) => {
  await loginAsAdmin(page);
  const campaignId = await fetchFirstCampaignId(page);
  if (!campaignId) {
    test.skip(true, 'integration: no campaigns for postback config');
    return;
  }

  const snapshotGet = page.waitForResponse(isApiGet('/api/v1/postbacks/snapshot'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/integrations/postbacks');
  await snapshotGet;

  await stubApiRoute(page, 'PUT', `/api/v1/postbacks/config/${campaignId}`, 400, {
    message: 'invalid postback url template',
  });

  await page.locator('#postback-campaign-id').fill(campaignId);
  await page.locator('#postback-provider').click();
  await page.getByRole('option', { name: 'webhook' }).click();
  await page
    .locator('#postback-url-template')
    .fill('https://example.com/postback?click={click_id}');
  await page.locator('#postback-target-event').fill('conversion');

  const savePut = page.waitForResponse(
    (response) =>
      response.request().method() === 'PUT' &&
      response.url().includes(`/api/v1/postbacks/config/${campaignId}`) &&
      response.status() === 400,
    { timeout: 20_000 }
  );
  await page.getByRole('button', { name: 'Save config' }).click();
  await savePut;

  await expect(mainContent(page).getByRole('alert')).toBeVisible({ timeout: 15_000 });
  await expect(
    mainContent(page).getByText('invalid postback url template', { exact: true })
  ).toBeVisible();
});
