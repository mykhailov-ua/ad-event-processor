import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  CATALOG_SPEND_VELOCITY,
  mockAuthedSession,
  mockEmptyCampaigns,
  mockReportCatalog,
} from './helpers.js';

test('main nav shows reports hub link without per-report sidebar clutter', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);
  await mockEmptyCampaigns(page);

  await page.goto('/campaigns');
  const nav = page.getByRole('navigation', { name: 'Main' });
  await expect(nav.getByRole('link', { name: 'Reports hub' })).toBeVisible();
  await expect(nav.getByRole('link', { name: 'Spend velocity' })).toHaveCount(0);
});

test('hamburger opens drawer, escape closes and returns focus', async ({ page }) => {
  await page.setViewportSize({ width: 600, height: 600 });
  await mockAuthedSession(page, ADMIN_USER);
  await mockEmptyCampaigns(page);

  await page.goto('/campaigns');
  const menu = page.getByRole('button', { name: 'Menu', exact: true });
  await expect(menu).toBeVisible();
  await menu.click();
  await expect(menu).toHaveAttribute('aria-expanded', 'true');

  await page.keyboard.press('Escape');
  await expect(menu).toHaveAttribute('aria-expanded', 'false');
  await expect(menu).toBeFocused();
});

test('reports hub catalog lists live report cards', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);
  await mockReportCatalog(page, [CATALOG_SPEND_VELOCITY]);

  await page.goto('/reports');
  await expect(page.getByTestId('reports-hub-page')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Spend velocity' })).toBeVisible();
});

test('telegram report route mounts runner chrome', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);

  const customerId = '00000000-0000-0000-0000-000000000001';
  const qs = `customer_id=${customerId}&from=2026-06-01&to=2026-08-01`;

  await page.route('**/api/v1/reports/telegram/summary*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        clicks: 1,
        impressions: 2,
        conversions: 0,
        premium: 0,
        motivated: 0,
        freshness: { state: 'ok' },
      }),
    });
  });

  await page.goto(`/reports/telegram?${qs}`);
  await expect(page.getByRole('heading', { name: /Telegram/i })).toBeVisible();
});
