import { apiFetch, apiJson } from './client.js';
import { reportKeyToApiPath } from '../lib/report_paths.js';
import type {
  ClickLogReportQuery,
  ClickLogReportResponse,
  FraudEvidencePack,
  ReportCatalogResponse,
  ReportJobSpec,
  ReportJobStatus,
  ReportMapEnvelope,
  ReportRunQuery,
  TelegramReportExportRequest,
} from './types.js';

export async function getReportCatalog(signal?: AbortSignal): Promise<ReportCatalogResponse> {
  return apiJson<ReportCatalogResponse>('/api/v1/reports/catalog', { signal });
}

export function buildReportRunPath(key: string, params: ReportRunQuery = {}): string {
  const search = new URLSearchParams();
  const basePath = reportKeyToApiPath(key);

  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  if (params.click_id) {
    search.set('click_id', params.click_id);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }

  const query = search.toString();
  return query ? `${basePath}?${query}` : basePath;
}

export async function runReport(
  key: string,
  params: ReportRunQuery = {},
  signal?: AbortSignal,
): Promise<ReportMapEnvelope> {
  return apiJson<ReportMapEnvelope>(buildReportRunPath(key, params), { signal });
}

export function buildClickLogReportPath(params: ClickLogReportQuery): string {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  if (params.click_id) {
    search.set('click_id', params.click_id);
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }
  return `/api/v1/reports/click-log?${search.toString()}`;
}

export async function getClickLogReport(
  params: ClickLogReportQuery,
  signal?: AbortSignal,
): Promise<ClickLogReportResponse> {
  return apiJson<ClickLogReportResponse>(buildClickLogReportPath(params), { signal });
}

export async function runEvidencePackReport(
  key: string,
  params: ReportRunQuery,
  signal?: AbortSignal,
): Promise<FraudEvidencePack> {
  return apiJson<FraudEvidencePack>(buildReportRunPath(key, params), { signal });
}

export async function createReportJob(
  spec: ReportJobSpec,
  signal?: AbortSignal,
): Promise<ReportJobStatus> {
  return apiJson<ReportJobStatus>('/api/v1/reports/jobs', {
    method: 'POST',
    body: JSON.stringify(spec),
    signal,
  });
}

export async function getReportJob(id: string, signal?: AbortSignal): Promise<ReportJobStatus> {
  return apiJson<ReportJobStatus>(`/api/v1/reports/jobs/${encodeURIComponent(id)}`, { signal });
}

export async function cancelReportJob(id: string, signal?: AbortSignal): Promise<ReportJobStatus> {
  return apiJson<ReportJobStatus>(`/api/v1/reports/jobs/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
}

export async function downloadReportJob(id: string, signal?: AbortSignal): Promise<Blob> {
  const response = await apiFetch(`/api/v1/reports/jobs/${encodeURIComponent(id)}/download`, {
    signal,
  });
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
  return response.blob();
}

export async function exportTelegramReport(
  body: TelegramReportExportRequest,
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/reports/telegram/export', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
