import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaign editor shows deep section headings', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const editLink = page.locator('main a[href$="/edit"]').first();
  const count = await editLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await editLink.click();

  const routingSection = page.getByText('Routing & ingress', { exact: true });
  const macroSection = page.getByText('Macro preview', { exact: true });
  await expect(routingSection.or(macroSection)).toBeVisible();
});
