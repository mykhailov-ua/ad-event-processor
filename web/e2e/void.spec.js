import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('void invoice requires confirm dialog accept', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices/inv-void-1**', async (route) => {
    if (route.request().url().includes('/deliveries')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0 }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: 'inv-void-1',
        customer_id: 'cust-1',
        status: 'open',
        total_micro: 1_000_000,
      }),
    });
  });

  let voided = false;
  await page.route('**/api/v1/billing/invoices/inv-void-1/void**', async (route) => {
    voided = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/billing/invoices/inv-void-1');
  await page.getByRole('button', { name: 'Void invoice' }).click();
  await expect.poll(() => voided).toBe(true);
});
