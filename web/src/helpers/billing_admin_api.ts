import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { apiBlob } from './api_blob.js';
import { getOrCreate } from './idempotency.js';
import { to } from '../lib/to.js';
import type {
  BillingExportCreateSpec,
  BillingExportJobDTO,
  BillingInvariantDTO,
  InvoiceDeliveryListResponse,
} from '../types/api/billing.js';

export type PollBillingExportOpts = {
  intervalMs?: number;
  maxAttempts?: number;
  signal?: AbortSignal;
};

/**
 * Pause until the given number of milliseconds elapse.
 */
function sleepMs(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new Error('aborted'));
      return;
    }
    const timer = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener('abort', () => {
        clearTimeout(timer);
        reject(new Error('aborted'));
      }, { once: true });
    }
  });
}

/**
 * Fetch billing ledger invariant for one customer or fleet-wide.
 */
export async function fetchBillingInvariant(customerId = ''): Promise<BillingInvariantDTO> {
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  const qs = params.toString();
  const path = qs ? `/api/v1/billing/invariant?${qs}` : '/api/v1/billing/invariant';
  const res = await api<BillingInvariantDTO>(path);
  return res.data ?? { ok: true };
}

/**
 * List invoice delivery attempts.
 */
export async function fetchInvoiceDeliveries(invoiceId: string): Promise<InvoiceDeliveryListResponse> {
  const res = await api<InvoiceDeliveryListResponse>(
    `/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}/deliveries`,
  );
  return res.data ?? { items: [] };
}

/**
 * Retry invoice email delivery.
 */
export async function retryInvoiceDelivery(invoiceId: string): Promise<void> {
  const scope = `invoice-delivery-retry:${invoiceId}`;
  await apiConfirmed(
    `/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}/deliveries/retry`,
    {
      method: 'POST',
      body: '{}',
      headers: { 'Idempotency-Key': getOrCreate(scope) },
      idempotencyScope: scope,
    },
  );
}

/**
 * Enqueue a billing ledger export job.
 */
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

/**
 * Poll billing export job until completed or failed.
 */
export async function pollBillingExportJob(
  jobId: string,
  opts: PollBillingExportOpts = {},
): Promise<BillingExportJobDTO> {
  const intervalMs = opts.intervalMs ?? 2000;
  const maxAttempts = opts.maxAttempts ?? 60;
  for (let i = 0; i < maxAttempts; i++) {
    if (opts.signal?.aborted) {
      throw new Error('aborted');
    }
    const [res, err] = await to(api<BillingExportJobDTO>(
      `/api/v1/billing/exports/${encodeURIComponent(jobId)}`,
      { signal: opts.signal },
    ));
    if (err) throw err;
    const status = String(res?.data?.status ?? '').toUpperCase();
    if (status === 'COMPLETED' || status === 'FAILED') {
      return res?.data as BillingExportJobDTO;
    }
    await sleepMs(intervalMs, opts.signal);
  }
  throw new Error('export job timed out');
}

/**
 * Download a completed billing export file.
 */
export async function downloadBillingExport(
  jobId: string,
  filename = 'ledger-export.csv',
  downloadUrl = '',
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
