import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaigns directory shows import validate controls', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await page.getByRole('button', { name: 'Import', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Import validate' })).toBeVisible();
  await expect(page.getByLabel('Validate job ID')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Poll job' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Import bundle' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Migrate import' })).toBeVisible();
});

test('campaign editor shows campaign tools tabs', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const editLink = page.locator('main a[href$="/edit"]').first();
  const count = await editLink.count();
  if (count === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await editLink.click();

  await expect(page.getByRole('heading', { name: 'Campaign tools' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Integration', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Fraud', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Ops', exact: true })).toBeVisible();
});
