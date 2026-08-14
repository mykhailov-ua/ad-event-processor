/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('overview shows KPI metrics and navigation', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ overall: 'ok', checks: [] }),
    });
  });

  await page.route('**/api/v1/ops/incidents', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ partial: false, shards: [], outbox: { pending: 0 } }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        outbox_pending: 3,
        rps_estimate: 1200,
        drift_alert: false,
        emergency_breaker: 'closed',
        services: [
          { name: 'Management', status: 'ok' },
          { name: 'ClickHouse', status: 'ok' },
        ],
      }),
    });
  });

  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();
  await expect(page.getByText('Outbox pending')).toBeVisible();
  await expect(page.getByText('1200')).toBeVisible();
  await expect(page.locator('#app-outlet').getByRole('link', { name: 'Campaigns' })).toBeVisible();
  await expect(page.locator('#app-outlet').getByRole('link', { name: 'Operations' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Doctor' })).toBeVisible();
  await expect(page.getByText('Management')).toBeVisible();
});
