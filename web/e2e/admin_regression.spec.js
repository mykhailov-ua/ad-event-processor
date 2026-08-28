import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('rtb deals list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/rtb/deals**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'rtb down' } }),
    });
  });

  await page.goto('/rtb/deals');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('rtb down')).toBeVisible();
});

test('ops recon list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/recon/runs**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'recon down' } }),
    });
  });

  await page.goto('/ops/recon');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('recon down')).toBeVisible();
});

test('audit export 500 surfaces toast not silent success', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/audit**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-Total-Count': '0',
      },
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/v1/audit/export**', async (route) => {
    await route.fulfill({
      status: 500,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'INTERNAL', message: 'export failed' } }),
    });
  });

  await page.goto('/audit');
  await page.getByRole('button', { name: 'Export CSV' }).click();
  await expect(page.getByText('Export failed')).toBeVisible();
});

test('billing invoice list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'billing down' } }),
    });
  });

  await page.goto('/billing');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('billing down')).toBeVisible();
});
