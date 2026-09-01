import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign editor compare section is visible', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const editLink = page.locator('main a[href$="/edit"]').first();
  const count = await editLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await editLink.click();
  await expect(page.getByRole('heading', { name: 'Compare campaigns' })).toBeVisible();
  await expect(page.getByLabel('Against campaign ID')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Compare' })).toBeVisible();
});
