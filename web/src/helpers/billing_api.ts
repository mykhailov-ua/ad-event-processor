import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type Invoice = {
  id?: string;
  customer_id?: string;
  billing_month?: string;
  total_micro?: number;
  status?: string;
  currency?: string;
};

export type InvoiceListResponse = {
  items?: Invoice[];
  total?: number;
  limit?: number;
  offset?: number;
};

export type BillingSummary = {
  invoiced_mtd_micro?: number;
  invoice_count_mtd?: number;
  undelivered_invoice_notifications?: number;
};

export type InvoiceListParams = {
  limit: number;
  offset: number;
  customer_id?: string;
  status?: string;
  month?: string;
  min_total?: number;
};

export function buildInvoicesListUrl(params: InvoiceListParams): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
  });
  if (params.customer_id) qs.set('customer_id', params.customer_id);
  if (params.status) qs.set('status', params.status);
  if (params.month) qs.set('month', params.month);
  if (params.min_total != null && Number.isFinite(params.min_total)) {
    qs.set('min_total', String(params.min_total));
  }
  return `/api/v1/billing/invoices?${qs.toString()}`;
}

export async function getBillingSummary(signal?: AbortSignal): Promise<BillingSummary> {
  const result = await api<BillingSummary>('/api/v1/billing/summary', { signal });
  return result.data ?? {};
}

export type InvoiceDetail = Invoice & {
  subtotal_micro?: number;
  tax_micro?: number;
  tax_scheme?: string;
  tax_rate_bps?: number;
  pdf_url?: string;
  lines?: Array<{
    description?: string;
    amount_micro?: number;
    quantity?: number;
  }>;
};

export type InvoiceLedgerLine = {
  id?: string;
  description?: string;
  amount_micro?: number;
  created_at?: string;
};

export type InvoiceLedgerLinesResponse = {
  items?: InvoiceLedgerLine[];
  next_cursor?: string;
};

export type InvoiceDelivery = {
  id?: string;
  status?: string;
  provider?: string;
  recipient?: string;
  retry_count?: number;
  created_at?: string;
};

export type InvoiceDeliveryListResponse = {
  items?: InvoiceDelivery[];
};

export type InvoiceDetailTab = 'header' | 'lines' | 'deliveries' | 'pdf';

export const INVOICE_DETAIL_TABS: Array<{ id: InvoiceDetailTab; label: string }> = [
  { id: 'header', label: 'Invoice' },
  { id: 'lines', label: 'Ledger lines' },
  { id: 'deliveries', label: 'Deliveries' },
  { id: 'pdf', label: 'PDF' },
];

export function parseInvoiceDetailTab(raw: string | null): InvoiceDetailTab {
  const allowed: InvoiceDetailTab[] = ['header', 'lines', 'deliveries', 'pdf'];
  return allowed.includes(raw as InvoiceDetailTab) ? (raw as InvoiceDetailTab) : 'header';
}

export async function fetchInvoice(id: string, signal?: AbortSignal): Promise<InvoiceDetail> {
  const result = await api<InvoiceDetail>(
    `/api/v1/billing/invoices/${encodeURIComponent(id)}`,
    { signal }
  );
  return result.data ?? {};
}

export async function fetchInvoiceLedgerLines(
  id: string,
  cursor?: string,
  limit = 50,
  signal?: AbortSignal
): Promise<InvoiceLedgerLinesResponse> {
  const qs = new URLSearchParams({ limit: String(limit) });
  if (cursor) qs.set('cursor', cursor);
  const result = await api<InvoiceLedgerLinesResponse>(
    `/api/v1/billing/invoices/${encodeURIComponent(id)}/ledger-lines?${qs.toString()}`,
    { signal }
  );
  return result.data ?? {};
}

export async function fetchInvoiceDeliveries(
  id: string,
  signal?: AbortSignal
): Promise<InvoiceDeliveryListResponse> {
  const result = await api<InvoiceDeliveryListResponse>(
    `/api/v1/billing/invoices/${encodeURIComponent(id)}/deliveries`,
    { signal }
  );
  return result.data ?? {};
}

export async function voidInvoice(id: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/billing/invoices/${encodeURIComponent(id)}/void`, {
    method: 'POST',
    body: '{}',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('void failed');
  }
}

export async function retryInvoiceDelivery(id: string): Promise<void> {
  const result = await apiConfirmed(
    `/api/v1/billing/invoices/${encodeURIComponent(id)}/deliveries/retry`,
    { method: 'POST', body: '{}' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('retry failed');
  }
}

export function invoicePdfUrl(id: string): string {
  return `/api/v1/billing/invoices/${encodeURIComponent(id)}/pdf`;
}
