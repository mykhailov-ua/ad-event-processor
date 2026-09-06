import { test, expect } from '@playwright/test';

import {
  applyCustomerScopeIfPrompted,
  expectApiListBoundToDom,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('self-serve invoices load from GET /api/v1/selfserve/invoices', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/portals');
  await expect(page.getByRole('heading', { name: 'Secondary portals' })).toBeVisible();

  const listResponse = page.waitForResponse(isApiGet('/api/v1/selfserve/invoices'), {
    timeout: 20_000,
  });
  await page.getByRole('link', { name: 'Self-serve', exact: true }).first().click();
  await expect(page.getByRole('heading', { name: 'Self-serve' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Billing statement' })).toBeVisible();
  await applyCustomerScopeIfPrompted(page);

  const response = await listResponse;
  const body = await response.json();
  expect(body).toHaveProperty('items');
  expect(Array.isArray(body.items)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No invoices',
    rowLabel: (row) => String(row.id ?? ''),
  });
});

test('publisher dashboard read contract on GET /api/v1/publisher/dashboard', async ({ page }) => {
  await loginAsAdmin(page);
  const dashboardResponse = page.waitForResponse(
    (response) =>
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/publisher/dashboard'),
    { timeout: 20_000 }
  );
  await gotoLive(page, '/publisher/dashboard');
  await expect(page.getByRole('heading', { name: 'Publisher dashboard' })).toBeVisible();

  const response = await dashboardResponse;
  if (response.status() === 403) {
    await expect(page.getByText('Could not load publisher dashboard')).toBeVisible({
      timeout: 15_000,
    });
    return;
  }

  expect(response.status()).toBe(200);
  const body = await response.json();
  if (body.kpis?.impressions != null) {
    await expect(page.getByText('Impressions', { exact: true })).toBeVisible({ timeout: 15_000 });
  }
});

test('report schedules load from GET /api/v1/report-schedules', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/report-schedules'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/report-schedules');
  await expect(page.getByRole('heading', { name: 'Report schedules' })).toBeVisible();
  await applyCustomerScopeIfPrompted(page);

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No schedules',
    rowLabel: (row) => String(row.report_key ?? row.id ?? ''),
  });
});
