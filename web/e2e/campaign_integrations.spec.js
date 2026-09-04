import { test, expect } from '@playwright/test';

import { openFirstCampaignEditor, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign editor integrations section is visible', async ({ page }) => {
  await loginAsAdmin(page);
  const campaignId = await openFirstCampaignEditor(page);
  if (!campaignId) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await expect(page.getByText('Integrations', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Traffic template ID')).toBeVisible();
});
