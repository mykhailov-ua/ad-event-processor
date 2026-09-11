import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  fetchSessionCustomerId,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('adops dashboard binds campaigns[] from GET /api/v1/dashboards/adops', async ({ page }) => {
  await loginAsAdmin(page);
  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  const adopsGet = page.waitForResponse(isApiGet('/api/v1/dashboards/adops'), { timeout: 20_000 });
  await gotoLive(page, `/dashboards/adops?customer_id=${customerId}`);
  const response = await adopsGet;
  expect(response.ok()).toBe(true);

  const body = await response.json();
  expect(Array.isArray(body.campaigns)).toBe(true);
  expect(body.customer_id).toBeTruthy();
  expect(body.period?.from).toBeTruthy();
  expect(body.period?.to).toBeTruthy();

  await expect(mainHeading(page, 'Ad ops dashboard')).toBeVisible();
  await expect(page.getByText('Could not load dashboard')).not.toBeVisible();

  if (body.campaigns.length === 0) {
    await expect(page.getByTestId('dashboard-adops-campaigns')).not.toBeVisible();
    return;
  }

  await expect(page.getByTestId('dashboard-adops-campaigns')).toBeVisible({ timeout: 15_000 });
  await expectApiListBoundToDom(page, body.campaigns, {
    emptyTitle: 'No campaigns',
    rowLabel: (row) => String(row.name ?? row.id ?? ''),
  });

  const firstStatus = body.campaigns[0]?.status;
  if (typeof firstStatus === 'string' && firstStatus.length > 0) {
    const statusLabel =
      firstStatus === 'ACTIVE'
        ? 'Active'
        : firstStatus === 'PAUSED'
          ? 'Paused'
          : firstStatus.charAt(0) + firstStatus.slice(1).toLowerCase();
    await expect(page.getByText(statusLabel, { exact: true }).first()).toBeVisible({
      timeout: 15_000,
    });
  }
});
