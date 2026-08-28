import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('cost sync view loads credentials panel', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/cost-sync/networks**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ network: 'meta', label: 'Meta' }]),
    });
  });

  await page.route('**/api/v1/cost-sync/credentials**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ network: 'meta', account_id: 'act-1', status: 'ok' }]),
    });
  });

  await page.route('**/api/v1/cost-sync/history**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([]),
    });
  });

  await page.goto('/integrations/cost-sync');
  await expect(page.getByTestId('cost-sync-view')).toBeVisible();
});

test('true roi report mounts runner', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/reports/true-roi**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ rows: [], freshness: { stale: false } }),
    });
  });

  await page.goto('/reports/true-roi?customer_id=cust-1');
  await expect(page.getByRole('heading', { name: 'True ROI' })).toBeVisible();
});
