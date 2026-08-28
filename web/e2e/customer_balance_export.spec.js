import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('customer balance tab shows wallet fields', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers/cust-1', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ id: 'cust-1', name: 'Acme', currency: 'USD' }),
    });
  });

  await page.route('**/api/v1/customers/cust-1/balance**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        balance: '50.00',
        currency: 'USD',
        ledger: [],
      }),
    });
  });

  await page.goto('/customers/cust-1?tab=balance');
  await expect(page.getByRole('tab', { name: 'Balance' })).toBeVisible();
  await expect(page.getByText('$50.00')).toBeVisible();
});
