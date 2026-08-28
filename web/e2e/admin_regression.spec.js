import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const RTB_USER = {
  ...ADMIN_USER,
  permissions: [...ADMIN_USER.permissions, 'rtb:read', 'rtb:write', 'settings:write'],
};

const AUDIT_USER = {
  ...ADMIN_USER,
  permissions: ['audit:read', 'settings:read'],
};

const RECON_USER = {
  ...ADMIN_USER,
  permissions: ['audit:read', 'shards:read'],
};

test('rtb deals list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, RTB_USER);

  await page.route('**/api/v1/rtb/deals**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'rtb store down' } }),
    });
  });

  await page.goto('/rtb/deals');
  await expect(page.getByText('Service unavailable')).toBeVisible();
  await expect(page.getByText('rtb store down')).toBeVisible();
  await expect(page.getByText('503')).toBeVisible();
});

test('ops recon list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, RECON_USER);

  await page.route('**/api/v1/recon/runs**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'recon unavailable' } }),
    });
  });

  await page.goto('/ops/recon');
  await expect(page.getByText('Service unavailable')).toBeVisible();
  await expect(page.getByText('recon unavailable')).toBeVisible();
});

test('audit export 500 surfaces toast not silent success', async ({ page }) => {
  await mockAuthedSession(page, AUDIT_USER);

  await page.route('**/api/v1/audit**', async (route) => {
    const url = route.request().url();
    if (url.includes('/export')) {
      await route.fulfill({
        status: 500,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ error: { code: 'INTERNAL', message: 'export failed' } }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json', 'X-Total-Count': '0' },
      body: '[]',
    });
  });

  await page.goto('/audit');
  await page.getByTestId('audit-export-csv').click();
  await expect(page.getByText('Internal error', { exact: true })).toBeVisible();
  await expect(page.getByText('internal error', { exact: true })).toBeVisible();
});

test('customer balance export 500 surfaces toast', async ({ page }) => {
  const customerId = '550e8400-e29b-41d4-a716-446655440000';
  await mockAuthedSession(page, ADMIN_USER);

  await page.route(`**/api/v1/customers/${customerId}`, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: customerId,
        name: 'Acme',
        status: 'active',
        created_at: '2026-01-01T00:00:00Z',
      }),
    });
  });

  await page.route('**/api/v1/campaigns?**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/wallet`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/tax-profile`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ country_code: 'US' }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/balance/export**`, async (route) => {
    await route.fulfill({
      status: 500,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'INTERNAL', message: 'balance export failed' } }),
    });
  });

  await page.goto(`/customers/${customerId}`);
  await page.getByTestId('customer-balance-export').click();
  await expect(page.getByText('Internal error', { exact: true })).toBeVisible();
  await expect(page.getByText('internal error', { exact: true })).toBeVisible();
});
