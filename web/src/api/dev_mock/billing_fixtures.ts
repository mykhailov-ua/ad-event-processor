import { DEV_MOCK_CUSTOMERS } from './fixtures.ts';
import {
  countDisplay,
  devMockIso,
  devMockMonth,
  microDisplay,
  parseLimitOffset,
  slicePage,
  usdToMicro,
} from './fixture_helpers.ts';
import { seedDeterministicUuid } from './seed_uuid.ts';

const INVOICE_STATUSES = ['finalized', 'draft', 'void'] as const;

function buildInvoices() {
  const invoices = [];
  for (let index = 0; index < 18; index += 1) {
    const customer = DEV_MOCK_CUSTOMERS[index % DEV_MOCK_CUSTOMERS.length];
    const subtotalMicro = usdToMicro(4200 + index * 317.5);
    const taxMicro = Math.floor(subtotalMicro * 0.08);
    const totalMicro = subtotalMicro + taxMicro;
    const monthOffset = -(index % 6);
    invoices.push({
      id: seedDeterministicUuid('invoice', index + 1),
      customer_id: customer.id,
      billing_month: devMockMonth(monthOffset),
      subtotal_micro: subtotalMicro,
      subtotal_micro_display: microDisplay(subtotalMicro),
      tax_micro: taxMicro,
      tax_micro_display: microDisplay(taxMicro),
      total_micro: totalMicro,
      total_micro_display: microDisplay(totalMicro),
      currency: 'USD',
      tax_scheme: 'US_SALES',
      tax_rate_bps: 800,
      status: INVOICE_STATUSES[index % INVOICE_STATUSES.length],
      created_at: devMockIso(index % 20),
      updated_at: devMockIso(index % 12, index),
    });
  }
  return invoices;
}

const DEV_MOCK_INVOICES = buildInvoices();

export function devMockBillingSummary() {
  const finalized = DEV_MOCK_INVOICES.filter((row) => row.status === 'finalized');
  const invoicedMtd = finalized.reduce((sum, row) => sum + row.total_micro, 0);
  const customerIds = new Set(finalized.map((row) => row.customer_id));
  return {
    invoiced_mtd_micro: invoicedMtd,
    invoiced_mtd_display: microDisplay(invoicedMtd),
    invoice_count_mtd: finalized.length,
    invoice_count_mtd_display: countDisplay(finalized.length),
    undelivered_invoice_notifications: 2,
    undelivered_invoice_notifications_display: '2',
    customers_with_spend_in_month: customerIds.size,
    customers_with_spend_in_month_display: countDisplay(customerIds.size),
  };
}

export function devMockInvoicesList(url: URL) {
  const { limit, offset } = parseLimitOffset(url);
  const month = url.searchParams.get('month')?.trim();
  const status = url.searchParams.get('status')?.trim();
  let filtered = DEV_MOCK_INVOICES;
  if (month) {
    filtered = filtered.filter((row) => row.billing_month === month);
  }
  if (status) {
    filtered = filtered.filter((row) => row.status === status);
  }
  const page = slicePage(filtered, limit, offset);
  return { ...page, limit, offset };
}

export function devMockInvoiceById(invoiceId: string) {
  return DEV_MOCK_INVOICES.find((row) => row.id === invoiceId);
}

export function devMockBillingInvariant(customerId: string | undefined) {
  const customer = customerId
    ? DEV_MOCK_CUSTOMERS.find((row) => row.id === customerId)
    : DEV_MOCK_CUSTOMERS[0];
  const balanceMicro = usdToMicro(18_420.55);
  return {
    ok: true,
    customer_id: customer?.id,
    balance_micro: balanceMicro,
    ledger_sum_micro: balanceMicro,
    diff_micro: 0,
    fleet_scan_limit: 500,
  };
}

export function devMockCustomerBalance(customerId: string) {
  const index = DEV_MOCK_CUSTOMERS.findIndex((row) => row.id === customerId);
  const balanceMicro = usdToMicro(12_000 + (index >= 0 ? index : 0) * 2_450.75);
  return {
    customer_id: customerId,
    balance_micro: balanceMicro,
    balance_micro_display: microDisplay(balanceMicro),
    currency: 'USD',
    updated_at: devMockIso(0, 2),
  };
}

export function devMockCustomerWallet(customerId: string) {
  const balance = devMockCustomerBalance(customerId);
  return {
    customer_id: customerId,
    currency: 'USD',
    available_micro: balance.balance_micro,
    reserved_micro: usdToMicro(250),
    updated_at: balance.updated_at,
  };
}

export function devMockCustomerLedger(customerId: string, url: URL) {
  const { limit, offset } = parseLimitOffset(url, 100);
  const entries = Array.from({ length: 24 }, (_, index) => {
    const debit = index % 3 !== 0;
    const amountMicro = usdToMicro(35 + index * 12.4);
    return {
      id: seedDeterministicUuid('ledger', index + 1),
      customer_id: customerId,
      campaign_id: seedDeterministicUuid('campaign', (index % 12) + 1),
      entry_type: debit ? 'debit' : 'credit',
      amount_micro: debit ? -amountMicro : amountMicro,
      amount_micro_display: microDisplay(debit ? -amountMicro : amountMicro),
      balance_after_micro: usdToMicro(12_000 - index * 40),
      description: debit ? 'Campaign spend sync' : 'Invoice payment',
      created_at: devMockIso(index % 30, index),
      created_at_display: devMockIso(index % 30, index),
    };
  });
  const page = slicePage(entries, limit, offset);
  return { ...page, limit, offset };
}

export function devMockCustomerStatement(customerId: string, url: URL) {
  const from = url.searchParams.get('from') ?? devMockIso(30);
  const to = url.searchParams.get('to') ?? devMockIso(0);
  const lines = Array.from({ length: 8 }, (_, index) => ({
    date: devMockIso(index * 3),
    description: index % 2 === 0 ? 'Media spend' : 'Adjustment',
    amount_micro: usdToMicro(index % 2 === 0 ? -(180 + index * 22) : 95 + index * 11),
    running_balance_micro: usdToMicro(10_000 - index * 120),
  }));
  return {
    customer_id: customerId,
    from,
    to,
    currency: 'USD',
    opening_balance_micro: usdToMicro(10_960),
    closing_balance_micro: usdToMicro(9_040),
    lines,
  };
}
