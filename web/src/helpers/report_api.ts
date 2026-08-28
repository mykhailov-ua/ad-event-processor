import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { downloadBlob, fetchBlob } from './api_blob.js';

export type ReportCatalogRow = {
  key?: string;
  title?: string;
  description?: string;
  category?: string;
  required_permissions?: string[];
  default_range?: string;
  export_formats?: string[];
  license_gated?: boolean;
  feature_key?: string;
};

export type ReportCatalogResponse = {
  rows: ReportCatalogRow[];
};

export type ReportJobSpec = {
  customer_id: string;
  report_key: string;
  from: string;
  to: string;
  format?: 'csv' | 'json';
};

export type ReportJobStatus = {
  id?: string;
  job_id?: string;
  customer_id?: string;
  report_key?: string;
  format?: string;
  status?: string;
  bytes?: number;
  error?: string;
  created_at?: string;
};

export type SavedView = {
  id?: string;
  owner_id?: string;
  owner_mask_level?: 'full' | 'masked';
  customer_id?: string;
  name?: string;
  report_key?: string;
  spec?: Record<string, unknown>;
  is_shared?: boolean;
  created_at?: string;
};

export type ReportQueryParams = {
  from?: string;
  to?: string;
  customer_id?: string;
  campaign_id?: string;
  limit?: number;
  offset?: number;
  cursor?: string;
};

export type ReportFreshness = {
  stale?: boolean;
  ch_lag_seconds?: number;
  lag_seconds?: number;
  as_of?: string;
  consistency?: string;
};

export type ReportFetchResponse = {
  rows?: Array<Record<string, unknown>>;
  freshness?: ReportFreshness;
  next_cursor?: string;
  total?: number;
  [key: string]: unknown;
};

export function reportApiPath(reportKey: string): string {
  switch (reportKey) {
    case 'click-log':
      return '/api/v1/reports/click-log';
    case 'rtb-overview':
      return '/api/v1/reports/rtb/overview';
    case 'rtb-no-bid-reasons':
      return '/api/v1/reports/rtb/no-bid-reasons';
    case 'rtb-geo-device':
      return '/api/v1/reports/rtb/geo-device';
    case 'telegram':
      return '/api/v1/reports/telegram/summary';
    case 'telegram-funnel':
      return '/api/v1/reports/telegram/funnel';
    case 'telegram-bots':
      return '/api/v1/reports/telegram/bots';
    case 'telegram-premium':
      return '/api/v1/reports/telegram/premium';
    case 'telegram-fraud':
      return '/api/v1/reports/telegram/fraud';
    case 'ghost-impression-funnel':
      return '/api/v1/reports/ghost-impression-funnel';
    default:
      return `/api/v1/reports/${reportKey}`;
  }
}

export function buildReportQueryString(params: ReportQueryParams): string {
  const qs = new URLSearchParams();
  if (params.from) qs.set('from', params.from);
  if (params.to) qs.set('to', params.to);
  if (params.customer_id) qs.set('customer_id', params.customer_id);
  if (params.campaign_id) qs.set('campaign_id', params.campaign_id);
  if (params.limit !== undefined) qs.set('limit', String(params.limit));
  if (params.offset !== undefined) qs.set('offset', String(params.offset));
  if (params.cursor) qs.set('cursor', params.cursor);
  return qs.toString();
}

export function buildReportUrl(reportKey: string, params: ReportQueryParams): string {
  const base = reportApiPath(reportKey);
  const query = buildReportQueryString(params);
  return query ? `${base}?${query}` : base;
}

export async function fetchReportCatalog(signal?: AbortSignal): Promise<ReportCatalogResponse> {
  const result = await api<ReportCatalogResponse>('/api/v1/reports/catalog', { signal });
  return result.data;
}

export async function fetchSavedViews(
  customerId: string | undefined,
  signal?: AbortSignal
): Promise<SavedView[]> {
  const qs = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const result = await api<{ items?: SavedView[]; rows?: SavedView[] }>(
    `/api/v1/views${qs}`,
    { signal }
  );
  const body = result.data;
  if (Array.isArray(body.items)) return body.items;
  if (Array.isArray(body.rows)) return body.rows;
  if (Array.isArray(body)) return body as SavedView[];
  return [];
}

export async function submitReportExportJob(
  spec: ReportJobSpec
): Promise<ReportJobStatus> {
  const result = await apiConfirmed<ReportJobStatus>('/api/v1/reports/jobs', {
    method: 'POST',
    body: JSON.stringify(spec),
  });
  return result.data;
}

export async function pollReportJob(jobId: string, signal?: AbortSignal): Promise<ReportJobStatus> {
  const result = await api<ReportJobStatus>(`/api/v1/reports/jobs/${encodeURIComponent(jobId)}`, {
    signal,
  });
  return result.data;
}

export async function downloadReportJobExport(
  jobId: string,
  filename: string
): Promise<void> {
  const result = await fetchBlob(`/api/v1/reports/jobs/${encodeURIComponent(jobId)}/download`);
  downloadBlob(result.blob, filename);
}

export function reportJobId(status: ReportJobStatus): string | null {
  return status.job_id ?? status.id ?? null;
}

export function isReportJobComplete(status: ReportJobStatus): boolean {
  const value = (status.status ?? '').toLowerCase();
  return value === 'complete' || value === 'completed' || value === 'done';
}

export function isReportJobFailed(status: ReportJobStatus): boolean {
  const value = (status.status ?? '').toLowerCase();
  return value === 'failed' || value === 'error' || value === 'cancelled';
}

export async function waitForReportJob(
  jobId: string,
  options: { signal?: AbortSignal; intervalMs?: number; maxAttempts?: number } = {}
): Promise<ReportJobStatus> {
  const intervalMs = options.intervalMs ?? 1500;
  const maxAttempts = options.maxAttempts ?? 120;
  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    if (options.signal?.aborted) {
      throw new DOMException('Aborted', 'AbortError');
    }
    const status = await pollReportJob(jobId, options.signal);
    if (isReportJobComplete(status) || isReportJobFailed(status)) {
      return status;
    }
    await new Promise<void>((resolve, reject) => {
      const timer = window.setTimeout(resolve, intervalMs);
      if (options.signal) {
        const onAbort = () => {
          window.clearTimeout(timer);
          reject(new DOMException('Aborted', 'AbortError'));
        };
        if (options.signal.aborted) {
          onAbort();
          return;
        }
        options.signal.addEventListener('abort', onAbort, { once: true });
        window.setTimeout(() => {
          options.signal?.removeEventListener('abort', onAbort);
        }, intervalMs + 50);
      }
    });
  }
  throw new Error('Report export job timed out');
}

export function extractReportRows(payload: ReportFetchResponse | null): Array<Record<string, unknown>> {
  if (!payload) return [];
  if (Array.isArray(payload.rows)) {
    return payload.rows as Array<Record<string, unknown>>;
  }
  return [];
}

export function extractReportFreshness(payload: ReportFetchResponse | null): ReportFreshness | null {
  if (!payload) return null;
  if (payload.freshness && typeof payload.freshness === 'object') {
    return payload.freshness as ReportFreshness;
  }
  return null;
}
