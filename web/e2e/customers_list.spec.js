/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('customers list shows rows from API', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers?**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [
          {
            id: 'cust-list-1',
            name: 'Acme Corp',
            balance: '1200.50',
            currency: 'USD',
            active_campaigns: 3,
            created_at: '2026-01-15T00:00:00Z',
          },
        ],
        total: 1,
      }),
    });
  });

  await page.goto('/customers');
  await expect(page.getByRole('heading', { name: 'Customers' })).toBeVisible();
  await expect(page.getByRole('cell', { name: 'Acme Corp' })).toBeVisible();
});

test('customers list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers?**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'customers store down' } }),
    });
  });

  await page.goto('/customers');
  await expect(page.getByText('Service unavailable')).toBeVisible();
  await expect(page.getByText('customers store down')).toBeVisible();
  await expect(page.getByText('503')).toBeVisible();
});
