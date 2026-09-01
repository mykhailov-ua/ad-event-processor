import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('portals hub and self-serve smoke on seed stack', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/portals');
  await expect(page.getByRole('heading', { name: 'Secondary portals' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Portals sections' })).toBeVisible();

  await page.getByRole('link', { name: 'Self-serve', exact: true }).first().click();
  await expect(page.getByRole('heading', { name: 'Self-serve' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Billing statement' })).toBeVisible();

  const customerRequired = page.getByText('Customer required', { exact: true });
  if (await customerRequired.isVisible({ timeout: 3000 }).catch(() => false)) {
    await page.getByRole('button', { name: 'Apply' }).click();
  }

  const invoicesTable = page.getByRole('table');
  const invoicesEmpty = page.getByText('No invoices', { exact: true });
  const forbidden = page.getByText('forbidden', { exact: false });
  await expect(invoicesTable.or(invoicesEmpty).or(forbidden)).toBeVisible({ timeout: 15_000 });
});

test('publisher portal smoke with permission-aware skip', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/publisher/dashboard');
  await expect(page.getByRole('heading', { name: 'Publisher dashboard' })).toBeVisible();

  const forbidden = page.getByText('forbidden', { exact: false });
  const unavailable = page.getByText('unavailable', { exact: false });
  const jsonCards = page.locator('[class*="grid"]').first();
  if (await forbidden.isVisible({ timeout: 5000 }).catch(() => false)) {
    test.skip(true, 'publisher dashboard forbidden for session permissions');
  }
  await expect(jsonCards.or(unavailable)).toBeVisible({ timeout: 15_000 });
});

test('report schedules portal smoke', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/report-schedules');
  await expect(page.getByRole('heading', { name: 'Report schedules' })).toBeVisible();

  const customerRequired = page.getByText('Customer required', { exact: true });
  if (await customerRequired.isVisible({ timeout: 3000 }).catch(() => false)) {
    await page.getByRole('button', { name: 'Apply' }).click();
  }

  const table = page.getByRole('table');
  const empty = page.getByText('No schedules', { exact: true });
  const forbidden = page.getByText('forbidden', { exact: false });
  await expect(table.or(empty).or(forbidden)).toBeVisible({ timeout: 15_000 });
});
