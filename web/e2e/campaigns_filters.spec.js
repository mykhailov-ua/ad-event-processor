import { test, expect } from '@playwright/test';

import { gotoCampaigns, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('campaigns directory toolbar and filters are visible', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await expect(page.getByRole('heading', { name: 'Campaigns' })).toBeVisible();
  await expect(page.getByRole('toolbar', { name: 'Campaign actions' })).toBeVisible();
  const toolbar = page.getByRole('toolbar', { name: 'Campaign actions' });
  await expect(page.getByRole('button', { name: 'Create', exact: true })).toBeVisible();
  await expect(toolbar.getByRole('button', { name: 'Clone' })).toBeVisible();
  await expect(toolbar.getByRole('button', { name: 'Pause', exact: true })).toBeVisible();
  await expect(toolbar.getByRole('button', { name: 'Archive', exact: true })).toBeVisible();
  await expect(page.getByLabel('Customer group')).toBeVisible();
  await expect(page.getByLabel('Pacing')).toBeVisible();
  await expect(page.getByLabel('Period')).toBeVisible();
  await expect(page.getByLabel('Search')).toBeVisible();
  await expect(page.getByRole('button', { name: /^Columns/ })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Export CSV' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Export JSON' })).toBeVisible();
});

test('campaigns status chip filter updates query string', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await page.getByRole('button', { name: /^Paused/ }).click();
  await expect(page).toHaveURL(/status=PAUSED/);

  await page
    .locator('[aria-label="Status and page summary"]')
    .getByRole('button', { name: /^All/ })
    .click();
  await expect(page).not.toHaveURL(/status=PAUSED/);
});

test('campaigns pacing filter updates query string', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await page.getByLabel('Pacing').click();
  await page.getByRole('option', { name: 'Even' }).click();
  await expect(page).toHaveURL(/pacing_mode=EVEN/);
});

test('campaigns search updates query string on enter', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const search = page.getByLabel('Search');
  await search.fill('alpha-campaign');
  await search.press('Enter');

  await expect(page).toHaveURL(/q=alpha-campaign/);
});

test('campaigns budget min filter updates query string on blur', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const budgetMin = page.getByLabel('Budget min ($)');
  await budgetMin.fill('10');
  await budgetMin.blur();

  await expect(page).toHaveURL(/budget_min_micro=10000000/);
});

test('campaigns create dialog opens from toolbar', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await page.getByRole('button', { name: 'Create', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Create campaign' })).toBeVisible();
});

test('campaigns column sort updates query string', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const nameSort = page.getByRole('button', { name: 'Name' });
  if ((await nameSort.count()) === 0) {
    test.skip(true, 'integration: campaigns table not rendered');
    return;
  }

  await nameSort.click();
  await expect(page).toHaveURL(/sort=name/);
});

test('campaigns selection clears when status filter changes', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const rowCheckbox = page.getByRole('checkbox', { name: /^Select / }).nth(1);
  if ((await rowCheckbox.count()) === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await rowCheckbox.check();
  await expect(page.getByRole('checkbox', { checked: true })).toHaveCount(1);

  await page.getByRole('button', { name: /^Paused/ }).click();
  await expect(page).toHaveURL(/status=PAUSED/);
  await expect(page.getByRole('checkbox', { checked: true })).toHaveCount(0);
});

test('campaigns pagination next updates offset when available', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  const nextButton = page.getByRole('button', { name: 'Next' });
  if (!(await nextButton.isEnabled())) {
    test.skip(true, 'integration: single page campaign list');
    return;
  }

  await nextButton.click();
  await expect(page).toHaveURL(/offset=/);
  const offset = new URL(page.url()).searchParams.get('offset');
  expect(Number(offset)).toBeGreaterThan(0);
});
