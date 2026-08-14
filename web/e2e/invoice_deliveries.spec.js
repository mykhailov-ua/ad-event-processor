/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const INVOICE_ID = 'inv-e2e-deliveries-1';
const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';

const OPEN_INVOICE = {
  id: INVOICE_ID,
  customer_id: CUSTOMER_ID,
  status: 'open',
  billing_month: '2026-07',
  total_micro: 1_000_000,
  tax_micro: 0,
  currency: 'USD',
};

const DELIVERIES = {
  items: [
    {
      id: 'del-1',
      status: 'FAILED',
      provider: 'smtp',
      recipient: 'billing@example.com',
      template_id: 'invoice_pdf',
      error_message: 'connection reset',
      retry_count: 2,
      created_at: '2026-08-12T09:00:00Z',
      updated_at: '2026-08-12T10:00:00Z',
    },
  ],
};

/**
 * @param {import('@playwright/test').Page} page
 * @param {object} invoice
 */
async function mockInvoiceDetail(page, invoice) {
  await page.route(`**/api/v1/billing/invoices/${INVOICE_ID}`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(invoice),
      });
      return;
    }
    await route.continue();
  });
}

test('invoice deliveries table and retry with confirm', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockInvoiceDetail(page, OPEN_INVOICE);

  let retryCalled = false;

  await page.route(`**/api/v1/billing/invoices/${INVOICE_ID}/deliveries`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(DELIVERIES),
      });
      return;
    }
    await route.continue();
  });

  await page.route(`**/api/v1/billing/invoices/${INVOICE_ID}/deliveries/retry`, async (route) => {
    retryCalled = true;
    await route.fulfill({ status: 202, body: '' });
  });

  await page.goto(`/billing/invoices/${INVOICE_ID}`);
  await expect(page.getByTestId('invoice-deliveries')).toBeVisible();
  await expect(page.getByText('connection reset')).toBeVisible();
  await expect(page.getByText('billing@example.com')).toBeVisible();

  await page.getByTestId('invoice-delivery-retry').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => retryCalled).toBe(true);
  await expect(page.getByText('Delivery retry queued')).toBeVisible();
});

test('invoice delivery retry hidden when invoice is void', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockInvoiceDetail(page, { ...OPEN_INVOICE, status: 'VOID' });

  await page.route(`**/api/v1/billing/invoices/${INVOICE_ID}/deliveries`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(DELIVERIES),
    });
  });

  await page.goto(`/billing/invoices/${INVOICE_ID}`);
  await expect(page.getByTestId('invoice-deliveries')).toBeVisible();
  await expect(page.getByTestId('invoice-delivery-retry')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Void' })).toHaveCount(0);
});
