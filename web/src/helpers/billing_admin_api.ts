import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { apiBlob } from './api_blob.js';
import { getOrCreate } from './idempotency.js';
import { to } from '../lib/to.js';
import type {
  BillingExportCreateSpec,
  BillingExportJobDTO,
  BillingForecastDTO,
  BillingInvariantDTO,
  BillingStatementDTO,
  DisputeListResponse,
  InvoiceDeliveryListResponse,
  InvoiceLedgerLinesResponse,
  InvoicePreviewDTO,
  BillingSummaryDTO,
  PaymentHistoryListResponse,
} from '../types/billing.js';

export type PollBillingExportOpts = {
  intervalMs?: number;
  maxAttempts?: number;
  signal?: AbortSignal;
};

function sleepMs(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new Error('aborted'));
      return;
    }
    const timer = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener(
        'abort',
        () => {
          clearTimeout(timer);
          reject(new Error('aborted'));
        },
        { once: true }
      );
    }
  });
}

export async function fetchBillingSummary(): Promise<BillingSummaryDTO> {
  const res = await api<BillingSummaryDTO>('/api/v1/billing/summary');
  return res.data ?? {};
}

export async function fetchBillingInvariant(customerId = ''): Promise<BillingInvariantDTO> {
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  const qs = params.toString();
  const path = qs ? `/api/v1/billing/invariant?${qs}` : '/api/v1/billing/invariant';
  const res = await api<BillingInvariantDTO>(path);
  return res.data ?? { ok: true };
}

export async function fetchInvoiceDeliveries(
  invoiceId: string
): Promise<InvoiceDeliveryListResponse> {
  const res = await api<InvoiceDeliveryListResponse>(
    `/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}/deliveries`
  );
  return res.data ?? { items: [] };
}

export async function retryInvoiceDelivery(invoiceId: string): Promise<void> {
  const scope = `invoice-delivery-retry:${invoiceId}`;
  await apiConfirmed(`/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}/deliveries/retry`, {
    method: 'POST',
    body: '{}',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    idempotencyScope: scope,
  });
}

export async function createBillingExport(spec: BillingExportCreateSpec): Promise<string> {
  const res = await apiConfirmed<{ job_id?: string }>('/api/v1/billing/exports', {
    method: 'POST',
    body: JSON.stringify(spec),
  });
  const jobId = (res.data as { job_id?: string })?.job_id;
  if (!jobId) {
    throw new Error('export job id missing from response');
  }
  return jobId;
}

export async function pollBillingExportJob(
  jobId: string,
  opts: PollBillingExportOpts = {}
): Promise<BillingExportJobDTO> {
  const intervalMs = opts.intervalMs ?? 2000;
  const maxAttempts = opts.maxAttempts ?? 60;
  for (let i = 0; i < maxAttempts; i++) {
    if (opts.signal?.aborted) {
      throw new Error('aborted');
    }
    const [res, err] = await to(
      api<BillingExportJobDTO>(`/api/v1/billing/exports/${encodeURIComponent(jobId)}`, {
        signal: opts.signal,
      })
    );
    if (err) throw err;
    const status = String(res?.data?.status ?? '').toUpperCase();
    if (status === 'COMPLETED' || status === 'FAILED') {
      return res?.data as BillingExportJobDTO;
    }
    await sleepMs(intervalMs, opts.signal);
  }
  throw new Error('export job timed out');
}

export async function downloadBillingExport(
  jobId: string,
  filename = 'ledger-export.csv',
  downloadUrl = ''
): Promise<void> {
  const path = downloadUrl.startsWith('/')
    ? downloadUrl
    : `/api/v1/billing/exports/${encodeURIComponent(jobId)}/download`;
  const blob = await apiBlob(path);
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export async function fetchBillingForecast(customerId: string): Promise<BillingForecastDTO> {
  const res = await api<BillingForecastDTO>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/billing/forecast`
  );
  return res.data ?? {};
}

export async function fetchDisputes(
  customerId: string,
  limit: number,
  offset: number
): Promise<DisputeListResponse> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  if (customerId) params.set('customer_id', customerId);
  const res = await api<DisputeListResponse>(`/api/v1/disputes?${params.toString()}`);
  return res.data ?? { disputes: [], total: 0 };
}

export async function fetchCustomerBillingStatement(
  customerId: string,
  month = ''
): Promise<BillingStatementDTO> {
  const params = new URLSearchParams();
  if (month) params.set('month', month);
  const qs = params.toString();
  const path = qs
    ? `/api/v1/customers/${encodeURIComponent(customerId)}/billing/statement?${qs}`
    : `/api/v1/customers/${encodeURIComponent(customerId)}/billing/statement`;
  const res = await api<BillingStatementDTO>(path);
  return res.data ?? {};
}

export async function previewBillingInvoice(
  customerId: string,
  billingMonth: string
): Promise<InvoicePreviewDTO> {
  const res = await apiConfirmed<InvoicePreviewDTO>('/api/v1/billing/invoices/preview', {
    method: 'POST',
    body: JSON.stringify({ customer_id: customerId, billing_month: billingMonth }),
  });
  return res.data ?? {};
}

export async function fetchCustomerPayments(
  customerId: string,
  limit = 20,
  offset = 0
): Promise<PaymentHistoryListResponse> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  const res = await api<PaymentHistoryListResponse>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/payments?${params.toString()}`
  );
  return res.data ?? { items: [], total: 0 };
}

export async function fetchInvoiceLedgerLines(
  invoiceId: string,
  cursor = '',
  limit = 50
): Promise<InvoiceLedgerLinesResponse> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (cursor) params.set('cursor', cursor);
  const res = await api<InvoiceLedgerLinesResponse>(
    `/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}/ledger-lines?${params.toString()}`
  );
  return res.data ?? { items: [], total: 0 };
}
