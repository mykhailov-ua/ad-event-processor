import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('dashboard buyer shows chart and breakdown sections', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboards/buyer');
  await expect(page.getByRole('heading', { name: /^dashboard$/i })).toBeVisible();

  const customer = page.getByLabel('Customer');
  await customer.click();
  await page.getByRole('option').nth(1).click();

  const apply = page.getByRole('button', { name: /^apply$/i });
  if (await apply.isVisible()) {
    await apply.click();
  }

  await expect(page.getByRole('region', { name: 'Key performance indicators' })).toBeVisible();
  await expect(page.getByRole('region', { name: 'Performance chart' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Campaigns', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Landing pages', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Recent clicks', exact: true })).toBeVisible();
  await expect(page.getByText('not available yet', { exact: false })).toHaveCount(0);
});
