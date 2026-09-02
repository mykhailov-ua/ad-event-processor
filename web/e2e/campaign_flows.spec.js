import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign directory create section opens from toolbar', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await expect(page.getByRole('heading', { name: 'Campaigns' })).toBeVisible();
  await page.getByRole('button', { name: 'Create campaign', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Create campaign' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Create' })).toBeVisible();

  const editLink = page.locator('main a[href$="/edit"]').first();
  const campaignCount = await editLink.count();
  if (campaignCount === 0) {
    return;
  }

  await expect(editLink).toBeVisible();
});
