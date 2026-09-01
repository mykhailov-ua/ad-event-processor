import { ApiError, apiFetch, apiJson } from './client.js';
import type {
  BillingForecast,
  BillingStatement,
  BillingSummary,
  CustomerBalance,
  CustomerLedgerListQuery,
  CustomerLedgerListResponse,
  CustomerPaymentsListQuery,
  InvoiceListQuery,
  InvoiceListResponse,
  PaymentHistoryListResponse,
  PreviewInvoiceRequest,
  InvoicePreview,
  InvoiceDeliveryListResponse,
  InvoiceLedgerLinesQuery,
  InvoiceLedgerLinesResponse,
  BillingInvariant,
  BillingInvariantQuery,
  BillingExportJobSpec,
  BillingExportJob,
  BillingExportJobCreatedResponse,
  TaxProfile,
  Wallet,
  Invoice,
} from './types.js';

export function buildInvoicesListPath(params: InvoiceListQuery = {}): string {
  const search = new URLSearchParams();

  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.month) {
    search.set('month', params.month);
  }
  if (params.status) {
    search.set('status', params.status);
  }
  if (params.min_total != null) {
    search.set('min_total', String(params.min_total));
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }

  const query = search.toString();
  return query ? `/api/v1/billing/invoices?${query}` : '/api/v1/billing/invoices';
}

export async function getBillingSummary(signal?: AbortSignal): Promise<BillingSummary> {
  return apiJson<BillingSummary>('/api/v1/billing/summary', { signal });
}

export async function listInvoices(
  params: InvoiceListQuery = {},
  signal?: AbortSignal,
): Promise<InvoiceListResponse> {
  return apiJson<InvoiceListResponse>(buildInvoicesListPath(params), { signal });
}

export async function getCustomerBillingStatement(
  customerId: string,
  month: string,
  signal?: AbortSignal,
): Promise<BillingStatement> {
  const search = new URLSearchParams({ month });
  return apiJson<BillingStatement>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/billing/statement?${search}`,
    { signal },
  );
}

export async function getCustomerBillingForecast(
  customerId: string,
  signal?: AbortSignal,
): Promise<BillingForecast> {
  return apiJson<BillingForecast>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/billing/forecast`,
    { signal },
  );
}

export async function getCustomerWallet(
  customerId: string,
  signal?: AbortSignal,
): Promise<Wallet> {
  return apiJson<Wallet>(`/api/v1/customers/${encodeURIComponent(customerId)}/wallet`, { signal });
}

export function buildCustomerPaymentsPath(
  customerId: string,
  params: CustomerPaymentsListQuery = {},
): string {
  const search = new URLSearchParams();

  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }

  const query = search.toString();
  const base = `/api/v1/customers/${encodeURIComponent(customerId)}/payments`;
  return query ? `${base}?${query}` : base;
}

export async function listCustomerPayments(
  customerId: string,
  params: CustomerPaymentsListQuery = {},
  signal?: AbortSignal,
): Promise<PaymentHistoryListResponse> {
  return apiJson<PaymentHistoryListResponse>(
    buildCustomerPaymentsPath(customerId, params),
    { signal },
  );
}

export async function getCustomerTaxProfile(
  customerId: string,
  signal?: AbortSignal,
): Promise<TaxProfile> {
  return apiJson<TaxProfile>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/tax-profile`,
    { signal },
  );
}

export async function putCustomerTaxProfile(
  customerId: string,
  body: TaxProfile,
  signal?: AbortSignal,
): Promise<TaxProfile> {
  return apiJson<TaxProfile>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/tax-profile`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function getCustomerBalance(
  customerId: string,
  signal?: AbortSignal,
): Promise<CustomerBalance> {
  return apiJson<CustomerBalance>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/balance`,
    { signal },
  );
}

export function buildCustomerLedgerPath(
  customerId: string,
  params: CustomerLedgerListQuery = {},
): string {
  const search = new URLSearchParams();

  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }

  const query = search.toString();
  const base = `/api/v1/customers/${encodeURIComponent(customerId)}/ledger`;
  return query ? `${base}?${query}` : base;
}

export async function listCustomerLedger(
  customerId: string,
  params: CustomerLedgerListQuery = {},
  signal?: AbortSignal,
): Promise<CustomerLedgerListResponse> {
  return apiJson<CustomerLedgerListResponse>(buildCustomerLedgerPath(customerId, params), {
    signal,
  });
}

export type CustomerBalanceExportResult = {
  blob: Blob;
};

export async function exportCustomerBalanceCsv(
  customerId: string,
  cursor?: string,
  signal?: AbortSignal,
): Promise<CustomerBalanceExportResult> {
  const search = new URLSearchParams({ format: 'csv' });
  if (cursor) {
    search.set('cursor', cursor);
  }

  const response = await apiFetch(
    `/api/v1/customers/${encodeURIComponent(customerId)}/balance/export?${search.toString()}`,
    { signal },
  );

  if (!response.ok) {
    let code = 'HTTP_ERROR';
    let message = response.statusText || `HTTP ${response.status}`;
    try {
      const body: unknown = await response.json();
      if (body && typeof body === 'object') {
        const record = body as Record<string, unknown>;
        const errorField = record.error;
        if (errorField && typeof errorField === 'object') {
          const errObj = errorField as Record<string, unknown>;
          if (typeof errObj.code === 'string') {
            code = errObj.code;
          }
          if (typeof errObj.message === 'string') {
            message = errObj.message;
          }
        }
      }
    } catch {
      // CSV or empty body on error.
    }
    throw new ApiError(response.status, code, message);
  }

  return { blob: await response.blob() };
}

export async function getBillingInvariant(
  params: BillingInvariantQuery = {},
  signal?: AbortSignal,
): Promise<BillingInvariant> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  const query = search.toString();
  const path = query ? `/api/v1/billing/invariant?${query}` : '/api/v1/billing/invariant';
  return apiJson<BillingInvariant>(path, { signal });
}

export async function getInvoice(id: string, signal?: AbortSignal): Promise<Invoice> {
  return apiJson<Invoice>(`/api/v1/billing/invoices/${encodeURIComponent(id)}`, { signal });
}

export async function downloadInvoicePdf(id: string, signal?: AbortSignal): Promise<Blob> {
  const response = await apiFetch(`/api/v1/billing/invoices/${encodeURIComponent(id)}/pdf`, {
    signal,
  });
  if (!response.ok) {
    throw new ApiError(response.status, 'HTTP_ERROR', response.statusText || `HTTP ${response.status}`);
  }
  return response.blob();
}

export function buildInvoiceLedgerLinesPath(
  id: string,
  params: InvoiceLedgerLinesQuery = {},
): string {
  const search = new URLSearchParams();
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  const query = search.toString();
  const base = `/api/v1/billing/invoices/${encodeURIComponent(id)}/ledger-lines`;
  return query ? `${base}?${query}` : base;
}

export async function listInvoiceLedgerLines(
  id: string,
  params: InvoiceLedgerLinesQuery = {},
  signal?: AbortSignal,
): Promise<InvoiceLedgerLinesResponse> {
  return apiJson<InvoiceLedgerLinesResponse>(buildInvoiceLedgerLinesPath(id, params), { signal });
}

export async function listInvoiceDeliveries(
  id: string,
  signal?: AbortSignal,
): Promise<InvoiceDeliveryListResponse> {
  return apiJson<InvoiceDeliveryListResponse>(
    `/api/v1/billing/invoices/${encodeURIComponent(id)}/deliveries`,
    { signal },
  );
}

export async function retryInvoiceDelivery(
  id: string,
  idempotencyKey: string,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<unknown>(
    `/api/v1/billing/invoices/${encodeURIComponent(id)}/deliveries/retry`,
    {
      method: 'POST',
      headers: { 'Idempotency-Key': idempotencyKey },
      body: JSON.stringify({}),
      signal,
    },
  );
}

export async function voidInvoice(id: string, signal?: AbortSignal): Promise<void> {
  await apiJson<unknown>(`/api/v1/billing/invoices/${encodeURIComponent(id)}/void`, {
    method: 'POST',
    signal,
  });
}

export async function previewInvoice(
  body: PreviewInvoiceRequest,
  signal?: AbortSignal,
): Promise<InvoicePreview> {
  return apiJson<InvoicePreview>('/api/v1/billing/invoices/preview', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function createBillingExportJob(
  body: BillingExportJobSpec,
  signal?: AbortSignal,
): Promise<BillingExportJobCreatedResponse> {
  return apiJson<BillingExportJobCreatedResponse>('/api/v1/billing/exports', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getBillingExportJob(
  jobId: string,
  signal?: AbortSignal,
): Promise<BillingExportJob> {
  return apiJson<BillingExportJob>(`/api/v1/billing/exports/${encodeURIComponent(jobId)}`, {
    signal,
  });
}

export async function downloadBillingExportJob(jobId: string, signal?: AbortSignal): Promise<Blob> {
  const response = await apiFetch(
    `/api/v1/billing/exports/${encodeURIComponent(jobId)}/download`,
    { signal },
  );
  if (!response.ok) {
    throw new ApiError(response.status, 'HTTP_ERROR', response.statusText || `HTTP ${response.status}`);
  }
  return response.blob();
}
