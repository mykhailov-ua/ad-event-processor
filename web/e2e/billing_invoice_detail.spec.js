import { test, expect } from '@playwright/test';

import {
  gotoBilling,
  isApiGet,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
  stubApiRoute,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('billing page shows ledger exports and operator tools', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  await expect(page.getByRole('link', { name: 'Ledger exports' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Ledger invariant' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Invoice preview' })).toBeVisible();
});

test('billing grid opens invoice detail when invoices exist', async ({ page }) => {
  await loginAsAdmin(page);
  const invoicesGet = page.waitForResponse(isApiGet('/api/v1/billing/invoices'), {
    timeout: 20_000,
  });
  await gotoBilling(page);
  const invoicesResponse = await invoicesGet;
  const invoicesBody = await invoicesResponse.json();
  const firstInvoice = invoicesBody.items?.[0];
  if (!firstInvoice?.id) {
    test.skip(true, 'integration: no invoices in billing grid');
    return;
  }

  const invoiceGet = page.waitForResponse(isApiGet(`/api/v1/billing/invoices/${firstInvoice.id}`), {
    timeout: 20_000,
  });
  const ledgerGet = page.waitForResponse(
    isApiGet(`/api/v1/billing/invoices/${firstInvoice.id}/ledger-lines`),
    { timeout: 20_000 }
  );
  const deliveriesGet = page.waitForResponse(
    isApiGet(`/api/v1/billing/invoices/${firstInvoice.id}/deliveries`),
    { timeout: 20_000 }
  );

  await page.locator(`main a[href="/billing/invoices/${firstInvoice.id}"]`).first().click();

  await invoiceGet;
  await ledgerGet;
  await deliveriesGet;

  await expect(page.getByRole('button', { name: 'Download PDF' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Ledger lines' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Deliveries' })).toBeVisible();
});

test('billing exports page loads from billing hub', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoBilling(page);

  await page.getByRole('link', { name: 'Ledger exports' }).click();
  await expect(page.getByRole('heading', { name: 'Billing ledger exports' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Start export' })).toBeVisible();
});

test('billing invoice detail GET 500 shows blocking error', { tag: '@L3' }, async ({ page }) => {
  await loginAsAdmin(page);
  const invoicesGet = page.waitForResponse(isApiGet('/api/v1/billing/invoices'), {
    timeout: 20_000,
  });
  await gotoBilling(page);
  const invoicesResponse = await invoicesGet;
  const invoicesBody = await invoicesResponse.json();
  const firstInvoice = invoicesBody.items?.[0];
  if (!firstInvoice?.id) {
    test.skip(true, 'integration: no invoices in billing grid');
    return;
  }

  await stubApiRoute(
    page,
    'GET',
    `/api/v1/billing/invoices/${firstInvoice.id}`,
    500,
    {
      code: 'INTERNAL_ERROR',
      message: 'invoice detail unavailable',
    },
    { exactPath: true }
  );

  await page.goto(`/billing/invoices/${firstInvoice.id}`);

  await expect(mainContent(page).getByRole('alert')).toBeVisible({ timeout: 15_000 });
  await expect(
    mainContent(page).getByText('Could not load invoice', { exact: true })
  ).toBeVisible();
  await expect(
    mainContent(page).getByText('The server encountered an error. Try again later.', {
      exact: true,
    })
  ).toBeVisible();
});
