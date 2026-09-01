import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaigns directory filter inputs are visible', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await expect(page.getByRole('heading', { name: 'Campaigns' })).toBeVisible();
  await expect(page.getByLabel('Customer')).toBeVisible();
  await expect(page.getByLabel('Status')).toBeVisible();
  await expect(page.getByLabel('Search')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Columns' })).toBeVisible();
});

test('campaigns status filter applies on Apply', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await page.getByLabel('Status').click();
  await page.getByRole('option', { name: 'Paused' }).click();
  await page.getByRole('button', { name: 'Apply' }).click();

  await expect(page).toHaveURL(/status=PAUSED/);
});
