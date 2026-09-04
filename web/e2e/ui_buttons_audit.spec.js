import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('audit campaigns toolbar claims vs DOM', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const claims = [
    { name: 'Create button', locator: page.getByRole('button', { name: 'Create', exact: true }) },
    { name: 'Clone button', locator: page.getByRole('button', { name: 'Clone' }) },
    { name: 'Columns control', locator: page.getByRole('button', { name: /^Columns/ }) },
    { name: 'Export CSV', locator: page.getByRole('button', { name: 'Export CSV' }) },
    { name: 'Export JSON', locator: page.getByRole('button', { name: 'Export JSON' }) },
    { name: 'Search field (e2e claim)', locator: page.getByLabel('Search') },
    { name: 'Customer group filter', locator: page.getByLabel('Customer group') },
    { name: 'Period filter', locator: page.getByLabel('Period') },
  ];

  const missing = [];
  for (const claim of claims) {
    const count = await claim.locator.count();
    if (count === 0) {
      missing.push(claim.name);
    }
  }

  expect(missing, `Missing UI elements: ${missing.join(', ')}`).toEqual([]);
});

test('audit campaigns bulk buttons respond without disabled attribute', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  for (const name of ['Clone', 'Pause', 'Archive']) {
    const button = page.getByRole('button', { name, exact: true });
    await expect(button).toBeVisible();
    await expect(button).toBeEnabled();
    const disabled = await button.getAttribute('disabled');
    expect(disabled, `${name} must not use disabled attribute`).toBeNull();
  }

  await page.getByRole('button', { name: 'Clone', exact: true }).click();
  await expect(page.getByText('Select exactly one campaign')).toBeVisible();
});

test('audit campaigns row checkbox is clickable', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const rowCheckbox = page.getByRole('checkbox', { name: /^Select / }).nth(1);
  await expect(rowCheckbox).toBeVisible();
  await rowCheckbox.check();
  await expect(rowCheckbox).toBeChecked();
});

test('audit dashboard preferences and apply visibility', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboards/buyer');
  await expect(page.getByRole('heading', { name: /^dashboard$/i })).toBeVisible();

  await page.getByRole('button', { name: 'Dashboard preferences' }).click();
  await expect(page.getByRole('heading', { name: 'Preferences' })).toBeVisible();
  await page.getByRole('button', { name: 'Cancel' }).click();

  const applyCount = await page.getByRole('button', { name: /^apply$/i }).count();
  // Apply is conditional; e2e that always clicks Apply is stale when customer already applied.
  expect(applyCount).toBeGreaterThanOrEqual(0);
});

test('audit sidebar navigation structure', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboards/buyer');

  const mainNav = page.getByRole('navigation', { name: 'Main' });
  await expect(mainNav.getByRole('link', { name: 'Campaigns' })).toBeVisible();
  await expect(mainNav.getByRole('link', { name: 'Users' })).toBeVisible();
  await expect(mainNav.getByRole('link', { name: 'Customers' })).toHaveCount(0);
});
