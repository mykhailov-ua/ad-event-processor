import { test, expect } from '@playwright/test';

import { gotoLive, isApiGet, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('edge parity report loads from GET /api/v1/reports/edge-parity', async ({ page }) => {
  await loginAsAdmin(page);
  const parityGet = page.waitForResponse(isApiGet('/api/v1/reports/edge-parity'), {
    timeout: 30_000,
  });
  await gotoLive(page, '/reports/edge-parity');
  await expect(page.getByRole('heading', { name: 'Edge parity' })).toBeVisible();

  const response = await parityGet;
  expect(response.ok()).toBe(true);
  const body = await response.json();
  expect(body).toHaveProperty('edge_ingress');
  expect(body).toHaveProperty('tracker_events');

  await expect(page.getByText('Edge ingress', { exact: true })).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText('Tracker events', { exact: true })).toBeVisible();
});
