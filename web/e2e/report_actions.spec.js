import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('source-quality report loads grid rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/reports/source-quality**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        rows: [{ source: 'meta', clicks: 10, conversions: 1 }],
        freshness: { stale: false },
      }),
    });
  });

  await page.goto('/reports/source-quality?customer_id=cust-1');
  await expect(page.getByRole('heading', { name: 'Source quality' })).toBeVisible();
  await expect(page.getByText('meta')).toBeVisible();
});

test('traffic-sources report accepts compare query param', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/reports/traffic-sources**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ rows: [], freshness: { stale: false } }),
    });
  });

  await page.goto('/reports/traffic-sources?customer_id=cust-1&compare=previous');
  await expect(page.getByRole('heading', { name: 'Traffic sources' })).toBeVisible();
  expect(page.url()).toContain('compare=previous');
});
