import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign directory create section opens from toolbar', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await expect(page.getByRole('heading', { name: 'Campaigns' })).toBeVisible();
  await page.getByRole('button', { name: 'Quick create', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Quick create campaign' })).toBeVisible();
  await expect(
    page.getByRole('dialog').getByRole('button', { name: 'Quick create', exact: true })
  ).toBeVisible();

  const editLink = page.locator('main a[href$="/edit"]').first();
  const campaignCount = await editLink.count();
  if (campaignCount === 0) {
    return;
  }

  await expect(editLink).toBeVisible();
});
