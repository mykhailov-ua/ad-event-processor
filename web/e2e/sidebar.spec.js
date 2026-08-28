import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('sidebar search focus stays inside sidebar at minimum width', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/eula', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ required: false }),
    });
  });

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.addInitScript(() => {
    localStorage.setItem('ui.sidebar.width', '220');
    localStorage.setItem('ui.sidebar.collapsed', '0');
  });

  await page.goto('/campaigns');
  const search = page.getByRole('combobox', { name: 'Search pages' });
  await search.focus();

  const overflow = await page.evaluate(() => {
    const sidebar = document.querySelector('.sidebar');
    const field = document.querySelector('.sidebar__search');
    if (!(sidebar instanceof HTMLElement) || !(field instanceof HTMLElement)) return null;
    const s = sidebar.getBoundingClientRect();
    const f = field.getBoundingClientRect();
    return Math.max(0, f.right - s.right, s.left - f.left);
  });

  expect(overflow).not.toBeNull();
  expect(overflow).toBeLessThanOrEqual(1);
});

test('sidebar search shows no dropdown chrome when idle', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/eula', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ required: false }),
    });
  });

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/campaigns');
  await expect(page.getByRole('combobox', { name: 'Search pages' })).toBeVisible();
  await expect(page.locator('.sidebar-search-dropdown')).toHaveCount(0);
});

test('sidebar search opens dropdown only after typing', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/eula', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ required: false }),
    });
  });

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/campaigns');
  const search = page.getByRole('combobox', { name: 'Search pages' });
  await search.focus();
  await expect(page.locator('.sidebar-search-dropdown')).toHaveCount(0);
  await search.fill('camp');
  await expect(page.locator('.sidebar-search-dropdown')).toHaveCount(1);
});

test('hamburger opens drawer, overlay closes and returns focus', async ({ page }) => {
  await page.setViewportSize({ width: 600, height: 600 });
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/campaigns');
  const menu = page.getByRole('button', { name: 'Menu' });
  await menu.click();
  await expect(menu).toHaveAttribute('aria-expanded', 'true');

  await page.keyboard.press('Escape');
  await expect(menu).toHaveAttribute('aria-expanded', 'false');
  await expect(menu).toBeFocused();
});

test('sidebar reports section shows hub and telegram only (P0 IA)', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/campaigns');
  const nav = page.getByRole('navigation');
  await expect(nav.getByRole('link', { name: 'Reports', exact: true })).toBeVisible();
  await expect(nav.getByRole('link', { name: 'Telegram Mini Apps' })).toBeVisible();
  await expect(nav.getByRole('link', { name: 'Spend velocity' })).toHaveCount(0);
  await expect(nav.getByRole('link', { name: 'Summary' })).toHaveCount(0);
});

test('reports hub catalog lists individual live reports', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/views*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [] }),
    });
  });

  await page.goto('/reports');
  await expect(page.getByRole('link', { name: 'Spend velocity' })).toBeVisible();
});

test('telegram analytics sub-nav uses in-page tabs only', async ({ page }) => {
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
  await page.route('**/api/v1/reports/telegram/funnel*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ rows: [], freshness: { state: 'ok' } }),
    });
  });
  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto(`/reports/telegram?${qs}`);
  await expect(
    page.getByRole('navigation', { name: 'Telegram reports' }).getByRole('link', { name: 'Funnel' })
  ).toBeVisible();
  await page
    .getByRole('navigation', { name: 'Telegram reports' })
    .getByRole('link', { name: 'Funnel' })
    .click();
  await expect(page).toHaveURL(/\/reports\/telegram\/funnel/);
  await expect(
    page.getByRole('navigation', { name: 'Telegram reports' }).getByRole('link', { name: 'Bots' })
  ).toBeVisible();
});
