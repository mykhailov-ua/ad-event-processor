import { test, expect } from '@playwright/test';

import {
  applyCampaignFilters,
  ensureLoggedIn,
  expectApiListBoundToDom,
  gotoCampaigns,
  gotoCampaignsLive,
  headerSearchInput,
  isCampaignsListResponse,
  navigateAppSection,
  skipUnlessIntegrationReady,
  statusFiltersBand,
} from './helpers.js';

test.describe.configure({ mode: 'serial', timeout: 120_000 });

test.beforeEach(async ({ page }, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
  await ensureLoggedIn(page);
});

test.afterEach(async ({ page }) => {
  await page.waitForTimeout(1500);
});

test('campaigns directory loads rows from GET /api/v1/campaigns', async ({ page }) => {
  const listResponse = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });
  await gotoCampaignsLive(page);
  const response = await listResponse;
  const body = await response.json();

  expect(body).toHaveProperty('items');
  expect(Array.isArray(body.items)).toBe(true);
  expect(body).toHaveProperty('total');
  expect(typeof body.total).toBe('number');

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No campaigns yet. Create one to start tracking spend and delivery.',
    rowLabel: (row) => String(row.name ?? ''),
  });
});

test('campaigns directory toolbar and filters are visible', async ({ page }) => {
  await gotoCampaigns(page);

  await expect(page.getByRole('heading', { name: 'Campaigns' })).toBeVisible();
  await expect(page.getByRole('toolbar', { name: 'Campaign actions' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Quick create', exact: true })).toBeVisible();
  await expect(page.getByLabel('Customer group')).toBeVisible();
  await expect(page.getByLabel('Pacing')).toBeVisible();
  await expect(page.getByLabel('Period')).toBeVisible();
  await expect(headerSearchInput(page)).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply', exact: true })).toBeVisible();
});

test('campaigns status chip filter updates query string', async ({ page }) => {
  await gotoCampaigns(page);

  await statusFiltersBand(page).getByRole('button', { name: /^Paused/ }).click();
  await applyCampaignFilters(page);
  await expect(page).toHaveURL(/status=PAUSED/);

  await statusFiltersBand(page).getByRole('button', { name: /^All/ }).click();
  await applyCampaignFilters(page);
  await expect(page).not.toHaveURL(/status=PAUSED/);
});

test('campaigns paused status chip applies PAUSED filter', async ({ page }) => {
  await gotoCampaigns(page);

  if (page.url().includes('status=PAUSED')) {
    await statusFiltersBand(page).getByRole('button', { name: /^All/ }).click();
    await applyCampaignFilters(page);
    await expect(page).not.toHaveURL(/status=PAUSED/);
  }

  await statusFiltersBand(page).getByRole('button', { name: /^Paused/ }).click();
  await applyCampaignFilters(page);
  await expect(page).toHaveURL(/status=PAUSED/);
  await expect(statusFiltersBand(page).getByRole('button', { name: /^Paused/ })).toHaveAttribute(
    'aria-pressed',
    'true'
  );
});

test('campaigns pacing select opens without render error', async ({ page }) => {
  await gotoCampaigns(page);

  await page.getByLabel('Pacing').click();
  await expect(page.getByRole('option', { name: 'Even' })).toBeVisible();
  await expect(page.getByText('PAGE ERROR')).toHaveCount(0);
});

test('campaigns header navigation works after opening pacing select', async ({ page }) => {
  await gotoCampaigns(page);

  await page.getByLabel('Pacing').click();
  await expect(page.getByRole('option', { name: 'Even' })).toBeVisible();
  await page.keyboard.press('Escape');

  await navigateAppSection(page, 'Customers');
  await expect(page).toHaveURL(/\/customers/);
  await expect(page.getByText('PAGE ERROR')).toHaveCount(0);
});

test('campaigns selection shows bulk actions beside Apply', async ({ page }) => {
  await gotoCampaigns(page);

  const rowCheckbox = page.getByRole('checkbox', { name: /^Select / }).first();
  if ((await rowCheckbox.count()) === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await rowCheckbox.click();
  await expect(page.getByRole('button', { name: 'Clone' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Pause', exact: true })).toBeVisible();
});

test('campaigns pacing filter updates query string', async ({ page }) => {
  await gotoCampaigns(page);

  await page.getByLabel('Pacing').click();
  await page.getByRole('option', { name: 'Even' }).click();
  await applyCampaignFilters(page);
  await expect(page).toHaveURL(/pacing_mode=EVEN/);
});

test('campaigns search updates query string on enter', async ({ page }) => {
  test.skip(true, 'integration: header search is command palette; campaigns q= is not wired there');
});

test('campaigns budget min filter updates query string on blur', async ({ page }) => {
  await gotoCampaigns(page);

  const budgetMin = page.getByLabel('Budget min ($)');
  await budgetMin.fill('10');
  await applyCampaignFilters(page);
  await expect(page).toHaveURL(/budget_min_micro=10000000/);
});

test('campaigns create dialog opens from toolbar', async ({ page }) => {
  await gotoCampaigns(page);

  await page.getByRole('button', { name: 'Quick create', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Quick create campaign' })).toBeVisible();
  await page.keyboard.press('Escape');
});

test('campaigns guided setup opens from toolbar menu', async ({ page }) => {
  await gotoCampaigns(page);

  await page.getByRole('button', { name: 'More campaign actions' }).click();
  await page.getByRole('menuitem', { name: 'Guided setup' }).click();
  await expect(page.getByRole('heading', { name: 'Guided setup' })).toBeVisible();
});

test('campaigns column sort updates query string', async ({ page }) => {
  await gotoCampaigns(page);

  const nameSort = page.getByRole('button', { name: 'Name' });
  if ((await nameSort.count()) === 0) {
    test.skip(true, 'integration: Name sort control not in campaigns table');
    return;
  }

  await nameSort.click();
  await expect(page).toHaveURL(/sort=name/);
});

test('campaigns selection clears when status filter changes', async ({ page }) => {
  await gotoCampaigns(page);

  const rowCheckbox = page.getByRole('checkbox', { name: /^Select / }).first();
  if ((await rowCheckbox.count()) === 0) {
    test.skip(true, 'integration: no campaigns in directory');
    return;
  }

  await rowCheckbox.click();
  await expect(page.getByRole('button', { name: 'Clone' })).toBeVisible();

  await statusFiltersBand(page).getByRole('button', { name: /^Paused/ }).click();
  await applyCampaignFilters(page);
  await expect(page).toHaveURL(/status=PAUSED/);
  await expect(page.getByRole('button', { name: 'Clone' })).toHaveCount(0);
});

test('campaigns pagination next updates offset when available', async ({ page }) => {
  await gotoCampaigns(page);

  const nextButton = page.getByRole('button', { name: 'Next' });
  if ((await nextButton.count()) === 0) {
    test.skip(true, 'integration: pagination controls not rendered');
    return;
  }
  if (!(await nextButton.isEnabled())) {
    test.skip(true, 'integration: single page campaign list');
    return;
  }

  await nextButton.click();
  await expect(page).toHaveURL(/offset=/);
  const offset = new URL(page.url()).searchParams.get('offset');
  expect(Number(offset)).toBeGreaterThan(0);
});
