import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';

test('customer detail balance CSV export', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let exportCalled = false;

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}`, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: CUSTOMER_ID,
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

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/wallet`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/tax-profile`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ country_code: 'US' }),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/balance/export**`, async (route) => {
    exportCalled = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'text/csv' },
      body: 'id,amount_micro,ledger_type,created_at\n1,1000,FEE,2026-01-01T00:00:00Z\n',
    });
  });

  await page.goto(`/customers/${CUSTOMER_ID}`);
  await expect(page.getByTestId('customer-balance-export')).toBeVisible();
  await page.getByTestId('customer-balance-export').click();
  await expect.poll(() => exportCalled).toBe(true);
});
