import { test, expect } from '@playwright/test';

import { gotoLive, isApiGet, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('dashboard buyer shows chart and breakdown sections after GET read', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/dashboards/buyer');
  await expect(page.getByRole('heading', { name: /^dashboard$/i })).toBeVisible();

  const customer = page.getByLabel('Customer');
  await customer.click();
  await page.getByRole('option').nth(1).click();

  const dashboardGet = page.waitForResponse(isApiGet('/api/v1/dashboards/buyer'), {
    timeout: 20_000,
  });

  const apply = page.getByRole('button', { name: /^apply$/i });
  if (await apply.isVisible()) {
    await apply.click();
  }

  const response = await dashboardGet;
  expect(response.ok()).toBe(true);

  await expect(page.getByRole('region', { name: 'Key performance indicators' })).toBeVisible();
  await expect(page.getByRole('region', { name: 'Performance chart' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Campaigns', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Landing pages', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Recent clicks', exact: true })).toBeVisible();
  await expect(page.getByText('not available yet', { exact: false })).toHaveCount(0);
});
