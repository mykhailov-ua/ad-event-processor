import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign editor integrations section is visible', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const editLink = page.locator('main a[href$="/edit"]').first();
  const count = await editLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await editLink.click();

  await expect(page.getByText('Integrations', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Traffic template ID')).toBeVisible();
});
