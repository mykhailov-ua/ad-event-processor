import { test, expect } from '@playwright/test';
import { BUYER_USER, mockAuthedSession } from './helpers.js';

test('self-serve billing statement loads summary', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/selfserve/billing/statement**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        opening_balance_micro: 0,
        closing_balance_micro: 5_000_000,
        lines: [],
      }),
    });
  });

  await page.route('**/api/v1/selfserve/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/selfserve/billing');
  await expect(page.getByTestId('selfserve-billing-panel')).toBeVisible();
  await expect(page.getByText('Closing')).toBeVisible();
});

test('buyer billing page locks customer filter', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/billing');
  await expect(page).toHaveURL(/customer_id=cust-1/);
});
