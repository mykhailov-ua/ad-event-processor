import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  BUYER_USER,
  installDialogAutoAccept,
  mockAuthedSession,
  mockEmptyCampaigns,
} from './helpers.js';

test('campaign list bulk pause is available when rows selected', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/campaigns\/[^/?]+/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [
          { id: 'c-1', name: 'One', status: 'active', customer_id: 'cust-1' },
          { id: 'c-2', name: 'Two', status: 'active', customer_id: 'cust-1' },
        ],
        total: 2,
      }),
    });
  });

  let bulkBody = null;
  await page.route('**/api/v1/campaigns/bulk**', async (route) => {
    bulkBody = route.request().postDataJSON();
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ updated: 2 }),
    });
  });

  await page.goto('/campaigns');
  await page.locator('input[type=checkbox]').first().check();
  await page.getByRole('region', { name: 'Bulk actions' }).getByRole('button', { name: 'Pause' }).click();
  expect(bulkBody?.action).toBe('pause');
});

test('buyer portfolio self-serve page loads', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/dashboards/buyer**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        customer_id: 'cust-1',
        active: 0,
        paused: 0,
        campaigns: [],
      }),
    });
  });

  await page.goto('/selfserve');
  await expect(page.getByTestId('selfserve-portfolio-panel')).toBeVisible();
});

test('live report campaign-overview does not mount stub banner', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/reports/campaign-overview**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ rows: [], freshness: { stale: false } }),
    });
  });

  await page.goto('/reports/campaign-overview?customer_id=cust-1');
  await expect(page.getByText('Dashboard unavailable')).toHaveCount(0);
});

test('operations hub reachable from shell', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockEmptyCampaigns(page);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0, rps_estimate: 0 }),
    });
  });

  await page.goto('/campaigns');
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Operations' }).click();
  await expect(page).toHaveURL(/\/ops/);
  await expect(page.getByRole('heading', { name: 'Operations' })).toBeVisible();
});
