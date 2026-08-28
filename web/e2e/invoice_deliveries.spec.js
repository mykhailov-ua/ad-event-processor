import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('invoice detail shows deliveries tab rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices/inv-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/deliveries')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          items: [{ id: 'del-1', provider: 'email', status: 'failed', recipient: 'a@test.local' }],
        }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: 'inv-1',
        customer_id: 'cust-1',
        status: 'open',
        total_micro: 5_000_000,
      }),
    });
  });

  await page.goto('/billing/invoices/inv-1?tab=deliveries');
  await expect(page.getByText('failed')).toBeVisible();
  await expect(page.getByText('a@test.local')).toBeVisible();
});
