/** harness=mock_api — disputes tab; does not prove Postgres payment_disputes seed. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const DISPUTES = {
  disputes: [
    {
      intent_id: 'pi-dispute-1',
      customer_id: 'cust-bill-1',
      amount_micro: 120_000_000,
      currency: 'USD',
      provider_dispute_id: 'dsp-e2e-01',
      updated_at: '2026-08-01T12:00:00Z',
      chargeback_ledger_entry_ids: [42],
    },
  ],
  total: 1,
};

test('billing disputes tab renders data row with id and amount', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/disputes**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(DISPUTES),
    });
  });

  await page.route('**/api/v1/customers/*/wallet', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.goto('/billing?tab=disputes&customer_id=cust-bill-1');
  await expect(page.getByTestId('disputes-table')).toBeVisible();
  const rows = page.locator('[data-testid="disputes-table"] tbody tr');
  await expect(rows).toHaveCount(1);
  await expect(page.getByTestId('dispute-id')).toContainText('dsp-e2e-01');
  await expect(page.locator('[data-testid="disputes-table"]')).toContainText('120.00');
});

test('billing forecast widget shows projected amount or empty state', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  const customerId = 'cust-forecast-1';

  await page.route(`**/api/v1/customers/${customerId}`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: customerId,
        name: 'Forecast Co',
        balance: '0.00',
        currency: 'USD',
      }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/wallet`, async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/billing/forecast`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        customer_id: customerId,
        month: '2026-08',
        ledger_mtd_micro: 45_000_000,
        ledger_run_rate_micro_per_day: 1_500_000,
        projected_month_end_micro: 90_000_000,
        days_remaining: 14,
        low_confidence: false,
      }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/campaigns**`, async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.route(`**/api/v1/customers/${customerId}/tax-profile`, async (route) => {
    await route.fulfill({ status: 404, body: '{}' });
  });

  await page.goto(`/customers/${customerId}`);
  await expect(page.getByTestId('billing-forecast-widget')).toBeVisible();
  await expect(page.getByTestId('forecast-projected')).toContainText('90.00');
});
