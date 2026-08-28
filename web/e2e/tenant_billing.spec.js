import { test, expect } from '@playwright/test';
import { BUYER_USER, mockAuthedSession } from './helpers.js';

test('role U billing locks customer_id', async ({ page }) => {
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
