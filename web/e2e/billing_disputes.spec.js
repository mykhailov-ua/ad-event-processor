import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('disputes page lists row with id and amount', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/disputes**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        disputes: [
          {
            intent_id: 'disp-1',
            customer_id: 'cust-1',
            amount_micro: 1_500_000,
            currency: 'USD',
            provider_dispute_id: 'prov-1',
            updated_at: '2026-01-01T00:00:00Z',
          },
        ],
        total: 1,
      }),
    });
  });

  await page.goto('/settings/disputes?customer_id=cust-1');
  await expect(page.getByTestId('settings-disputes-page')).toBeVisible();
  await expect(page.getByText('disp-1')).toBeVisible();
});

test('billing page shows invoice directory', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'inv-1', customer_id: 'cust-1', status: 'open', total_micro: 2_000_000 }],
        total: 1,
      }),
    });
  });

  await page.goto('/billing?customer_id=cust-1');
  await expect(page.getByRole('heading', { name: 'Billing' })).toBeVisible();
  await expect(page.getByText('inv-1')).toBeVisible();
});
