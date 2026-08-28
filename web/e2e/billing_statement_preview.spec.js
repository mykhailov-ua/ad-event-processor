import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('customer detail overview shows name', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers/cust-1', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ id: 'cust-1', name: 'Acme', currency: 'USD' }),
    });
  });

  await page.goto('/customers/cust-1');
  await expect(page.getByRole('heading', { name: 'Acme' })).toBeVisible();
});

test('billing page lists invoices for customer filter', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'inv-99', customer_id: 'cust-1', status: 'paid', total_micro: 10_000_000 }],
        total: 1,
      }),
    });
  });

  await page.goto('/billing?customer_id=cust-1');
  await expect(page.getByText('inv-99')).toBeVisible();
});
