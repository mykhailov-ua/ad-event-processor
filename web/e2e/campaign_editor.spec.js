import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign editor opens from directory', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await expect(page.getByRole('heading', { name: 'Campaigns' })).toBeVisible();

  const editLink = page.locator('main a[href$="/edit"]').first();
  const count = await editLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await editLink.click();
  await expect(page.getByRole('heading', { name: 'Campaign settings' })).toBeVisible();
  await expect(page.getByLabel('Name')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save' })).toBeVisible();
});
