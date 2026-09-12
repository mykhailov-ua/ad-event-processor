import { test, expect } from '@playwright/test';

import {
  integrationCampaignValidateSuffix,
  integrationRunToken,
  loginAsAdmin,
  openFirstCampaignEditor,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'campaign editor publish gate validates draft and checks publish readiness',
  { tag: '@write' },
  async ({ page }) => {
    await loginAsAdmin(page);

    const campaignId = await openFirstCampaignEditor(page);
    if (!campaignId) {
      test.skip(true, 'integration: no campaigns in directory');
      return;
    }

    await expect(page.getByRole('heading', { name: 'Publish gate' })).toBeVisible();

    const runToken = integrationRunToken();
    const nameField = page.getByLabel('Name');
    const currentName = await nameField.inputValue();
    await nameField.fill(`${currentName}${integrationCampaignValidateSuffix(runToken)}`);

    const validatePost = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        response.url().includes(`/api/v1/campaigns/${campaignId}/validate`) &&
        response.status() === 200,
      { timeout: 20_000 }
    );

    await page.getByRole('button', { name: 'Validate changes' }).click();

    const validateResponse = await validatePost;
    const validateBody = await validateResponse.json();
    expect(validateBody).toHaveProperty('valid');

    await expect(page.getByText('Patch validation', { exact: true })).toBeVisible({
      timeout: 15_000,
    });
    await expect(
      page.getByText(validateBody.valid ? 'Valid' : 'Invalid', { exact: true })
    ).toBeVisible();

    const publishCheckGet = page.waitForResponse(
      (response) =>
        response.request().method() === 'GET' &&
        response.url().includes(`/api/v1/campaigns/${campaignId}/publish-check`) &&
        response.status() === 200,
      { timeout: 20_000 }
    );

    await page.getByRole('button', { name: 'Check publish' }).click();

    const publishCheckResponse = await publishCheckGet;
    const publishCheckBody = await publishCheckResponse.json();
    expect(publishCheckBody).toHaveProperty('valid');

    await expect(page.getByText('Publish check', { exact: true })).toBeVisible({
      timeout: 15_000,
    });
    await expect(
      page.getByText(publishCheckBody.valid ? 'Ready' : 'Blocked', { exact: true })
    ).toBeVisible();
  }
);

test(
  'campaign editor publish check shows unavailable stub on 501',
  { tag: '@write' },
  async ({ page }) => {
    await loginAsAdmin(page);

    const campaignId = await openFirstCampaignEditor(page);
    if (!campaignId) {
      test.skip(true, 'integration: no campaigns in directory');
      return;
    }

    await page.route(`**/api/v1/campaigns/${campaignId}/publish-check`, async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 501,
        contentType: 'application/json',
        body: JSON.stringify({
          error: { code: 'NOT_IMPLEMENTED', message: 'publish check unavailable (e2e stub)' },
        }),
      });
    });

    const publishCheckGet = page.waitForResponse(
      (response) =>
        response.request().method() === 'GET' &&
        response.url().includes(`/api/v1/campaigns/${campaignId}/publish-check`) &&
        response.status() === 501,
      { timeout: 20_000 }
    );

    await page.getByRole('button', { name: 'Check publish' }).click();

    await publishCheckGet;
    await expect(page.getByText('publish check unavailable (e2e stub)')).toBeVisible({
      timeout: 15_000,
    });
  }
);
