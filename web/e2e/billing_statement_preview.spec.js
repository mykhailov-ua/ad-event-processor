import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = 'cust-bill-p5';

const STATEMENT = {
  customer_id: CUSTOMER_ID,
  period: { from: '2026-08-01T00:00:00Z', to: '2026-09-01T00:00:00Z' },
  opening_balance_micro: 50_000_000,
  closing_balance_micro: 35_000_000,
  currency: 'USD',
  reconciliation: { invoice_total_micro: 15_000_000, ledger_sum_micro: 15_000_000, delta_micro: 0 },
  invoices: [
    {
      id: 'inv-stmt-1',
      billing_month: '2026-08',
      status: 'ISSUED',
      total_micro: 15_000_000,
      currency: 'USD',
    },
  ],
};

const PREVIEW = {
  customer_id: CUSTOMER_ID,
  billing_month: '2026-08',
  currency: 'USD',
  subtotal_micro: 12_000_000,
  tax_micro: 1_200_000,
  total_micro: 13_200_000,
  lines: [{ ledger_type: 'SPEND', amount_micro: 12_000_000, entry_count: 4 }],
  would_skip: false,
};

const PAYMENTS = {
  items: [
    {
      intent_id: 'pi-p5-1',
      customer_id: CUSTOMER_ID,
      amount_micro: 100_000_000,
      currency: 'USD',
      status: 'PAYMENT_INTENT_STATUS_SUCCEEDED',
      provider: 'stripe',
      created_at: '2026-08-10T12:00:00Z',
    },
  ],
  total: 1,
  limit: 10,
  offset: 0,
};

test('customer detail shows billing statement and payments', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: CUSTOMER_ID,
        name: 'Bill Co',
        balance: '50.00',
        currency: 'USD',
      }),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/wallet`, async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ balance_micro: 50_000_000, currency: 'USD' }),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/billing/forecast`, async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ customer_id: CUSTOMER_ID, projected_month_end_micro: 40_000_000 }),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/billing/statement**`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(STATEMENT),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/payments**`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(PAYMENTS),
    });
  });

  await page.route(`**/api/v1/campaigns?**`, async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto(`/customers/${CUSTOMER_ID}`);
  await page.getByTestId('billing-statement-load').click();
  await expect(page.getByTestId('billing-statement-result')).toContainText('50.00');
  await expect(page.getByTestId('billing-statement-result')).toContainText('35.00');
  await expect(page.getByTestId('billing-payments-table')).toContainText('pi-p5-1');
  await expect(page.getByTestId('billing-payments-table')).toContainText('100.00');
});

test('billing invoices tab previews invoice totals', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let previewPosted = false;

  await page.route('**/api/v1/billing/invoices/preview', async (route) => {
    previewPosted = route.request().method() === 'POST';
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(PREVIEW),
    });
  });

  await page.route('**/api/v1/billing/invoices?**', async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.route('**/api/v1/customers/*/wallet', async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.goto(`/billing?tab=invoices&customer_id=${CUSTOMER_ID}`);
  await page.getByTestId('invoice-preview-submit').click();
  await expect.poll(() => previewPosted).toBe(true);
  await expect(page.getByTestId('invoice-preview-total')).toContainText('13.20');
});
