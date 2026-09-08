import { apiFetch, apiJson, parseApiError } from './client.js';
import { reportKeyToApiPath } from '../lib/report_paths.js';
import type {
  ClickLogReportQuery,
  ClickLogReportResponse,
  CustomerFraudByTypeReportResponse,
  FraudBreakdownReportResponse,
  FraudEvidencePack,
  FraudReasonRow,
  FraudReasonsReportKey,
  ReportCatalogResponse,
  ReportJobSpec,
  ReportJobStatus,
  ReportMapEnvelope,
  PostbackReconReportResponse,
  PlacementReportResponse,
  KeywordReportResponse,
  GeoROIReportResponse,
  TrafficSourcesReportResponse,
  DataQualityReportResponse,
  PacingDriftReportResponse,
  ConversionTypePayoutReportResponse,
  RtbOverviewReportResponse,
  RtbNoBidReasonsReportResponse,
  RtbGeoDeviceReportResponse,
  RtbReportQuery,
  CustomerScopedReportQuery,
  SourceQualityReportResponse,
  ReportRunQuery,
  TelegramReportExportRequest,
  TelegramReportExportResponse,
  WireSignalBreakdownReportResponse,
} from './types.js';
import type {
  CampaignToggleCohortQuery,
  CampaignToggleCohortReportResponse,
  FraudCatalogReportKey,
  FraudCatalogReportQuery,
  FraudCatalogReportResponse,
  PostbackReconciliationQuery,
  SourceQualityReportQuery,
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

export function buildFraudCatalogReportPath(
  key: FraudCatalogReportKey,
  params: FraudCatalogReportQuery = {}
): string {
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
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  if (params.dimension) {
    search.set('dimension', params.dimension);
  }
  if (params.compare) {
    search.set('compare', '1');
  }
  if (params.slice) {
    search.set('slice', '1');
  }
  if (params.layer_desync_count != null) {
    search.set('layer_desync_count', String(params.layer_desync_count));
  }

  const query = search.toString();
  return query ? `${basePath}?${query}` : basePath;
}

export async function getFraudCatalogReport(
  key: FraudCatalogReportKey,
  params: FraudCatalogReportQuery = {},
  signal?: AbortSignal
): Promise<FraudCatalogReportResponse> {
  return apiJson<FraudCatalogReportResponse>(buildFraudCatalogReportPath(key, params), { signal });
}

export function buildCampaignToggleCohortPath(
  params: CampaignToggleCohortQuery
): string {
  const search = new URLSearchParams();
  search.set('campaign_id', params.campaign_id);
  search.set('toggle_field', params.toggle_field);
  if (params.toggle_at) {
    search.set('toggle_at', params.toggle_at);
  }
  if (params.window_hours != null) {
    search.set('window_hours', String(params.window_hours));
  }
  return `/api/v1/reports/campaign-toggle-cohort?${search.toString()}`;
}

export async function getCampaignToggleCohortReport(
  params: CampaignToggleCohortQuery,
  signal?: AbortSignal
): Promise<CampaignToggleCohortReportResponse> {
  return apiJson<CampaignToggleCohortReportResponse>(buildCampaignToggleCohortPath(params), {
    signal,
  });
}

export async function runReport(
  key: string,
  params: ReportRunQuery = {},
  signal?: AbortSignal
): Promise<ReportMapEnvelope> {
  return apiJson<ReportMapEnvelope>(buildReportRunPath(key, params), { signal });
}

export type FraudReasonsReportResponse = {
  rows: FraudReasonRow[];
  freshness?: WireSignalBreakdownReportResponse['freshness'];
  next_cursor?: string;
};

export async function getFraudReasonsReport(
  reportKey: FraudReasonsReportKey,
  params: ReportRunQuery = {},
  signal?: AbortSignal
): Promise<FraudReasonsReportResponse> {
  const path = buildReportRunPath(reportKey, params);
  if (reportKey === 'wire-signal-breakdown') {
    const payload = await apiJson<WireSignalBreakdownReportResponse>(path, { signal });
    return {
      rows: payload.rows ?? [],
      freshness: payload.freshness,
      next_cursor: payload.next_cursor,
    };
  }
  const payload = await apiJson<FraudBreakdownReportResponse>(path, { signal });
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

export async function getCustomerFraudByTypeReport(
  params: ReportRunQuery = {},
  signal?: AbortSignal
): Promise<CustomerFraudByTypeReportResponse> {
  return apiJson<CustomerFraudByTypeReportResponse>(
    buildReportRunPath('customer-fraud-by-type', params),
    { signal }
  );
}

export function buildSourceQualityReportPath(params: SourceQualityReportQuery = {}): string {
  const search = new URLSearchParams();
  const basePath = reportKeyToApiPath('source-quality');

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
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }
  if (params.compare) {
    search.set('compare', '1');
  }
  for (const dim of params.group_by ?? []) {
    search.append('group_by', dim);
  }

  const query = search.toString();
  return query ? `${basePath}?${query}` : basePath;
}

export async function getSourceQualityReport(
  params: SourceQualityReportQuery = {},
  signal?: AbortSignal
): Promise<SourceQualityReportResponse> {
  return apiJson<SourceQualityReportResponse>(buildSourceQualityReportPath(params), { signal });
}

export async function getPostbackReconciliationReport(
  params: PostbackReconciliationQuery = {},
  signal?: AbortSignal
): Promise<PostbackReconReportResponse> {
  return apiJson<PostbackReconReportResponse>(
    buildReportRunPath('postback-reconciliation', params),
    { signal }
  );
}

export function buildCustomerReportPath(key: string, params: CustomerScopedReportQuery = {}): string {
  const path = buildReportRunPath(key, {
    customer_id: params.customer_id,
    from: params.from,
    to: params.to,
    campaign_id: params.campaign_id,
    limit: params.limit,
    offset: params.offset,
    cursor: params.cursor,
  });
  if (!params.compare) {
    return path;
  }
  const [base, query = ''] = path.split('?');
  const search = new URLSearchParams(query);
  search.set('compare', '1');
  return `${base}?${search.toString()}`;
}

export async function getPlacementsReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<PlacementReportResponse> {
  return apiJson<PlacementReportResponse>(buildCustomerReportPath('placements', params), { signal });
}

export async function getKeywordsReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<KeywordReportResponse> {
  return apiJson<KeywordReportResponse>(buildCustomerReportPath('keywords', params), { signal });
}

export async function getGeoRoiReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<GeoROIReportResponse> {
  return apiJson<GeoROIReportResponse>(buildCustomerReportPath('geo-roi', params), { signal });
}

export async function getTrafficSourcesReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<TrafficSourcesReportResponse> {
  return apiJson<TrafficSourcesReportResponse>(
    buildCustomerReportPath('traffic-sources', params),
    { signal }
  );
}

export async function getDataQualityReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<DataQualityReportResponse> {
  return apiJson<DataQualityReportResponse>(
    buildCustomerReportPath('data-quality', params),
    { signal }
  );
}

export async function getPacingDriftReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<PacingDriftReportResponse> {
  return apiJson<PacingDriftReportResponse>(
    buildCustomerReportPath('pacing-drift', params),
    { signal }
  );
}

export async function getConversionTypePayoutReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<ConversionTypePayoutReportResponse> {
  return apiJson<ConversionTypePayoutReportResponse>(
    buildCustomerReportPath('conversion-type-payout', params),
    { signal }
  );
}

export async function getRtbOverviewReport(
  params: RtbReportQuery = {},
  signal?: AbortSignal
): Promise<RtbOverviewReportResponse> {
  return apiJson<RtbOverviewReportResponse>(
    buildReportRunPath('rtb-overview', params),
    { signal }
  );
}

export async function getRtbNoBidReasonsReport(
  params: RtbReportQuery = {},
  signal?: AbortSignal
): Promise<RtbNoBidReasonsReportResponse> {
  return apiJson<RtbNoBidReasonsReportResponse>(
    buildReportRunPath('rtb-no-bid-reasons', params),
    { signal }
  );
}

export async function getRtbGeoDeviceReport(
  params: RtbReportQuery = {},
  signal?: AbortSignal
): Promise<RtbGeoDeviceReportResponse> {
  return apiJson<RtbGeoDeviceReportResponse>(
    buildReportRunPath('rtb-geo-device', params),
    { signal }
  );
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
  signal?: AbortSignal
): Promise<ClickLogReportResponse> {
  return apiJson<ClickLogReportResponse>(buildClickLogReportPath(params), { signal });
}

export async function runEvidencePackReport(
  key: string,
  params: ReportRunQuery,
  signal?: AbortSignal
): Promise<FraudEvidencePack> {
  return apiJson<FraudEvidencePack>(buildReportRunPath(key, params), { signal });
}

export async function createReportJob(
  spec: ReportJobSpec,
  signal?: AbortSignal
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
    throw await parseApiError(response);
  }
  return response.blob();
}

export async function exportTelegramReport(
  body: TelegramReportExportRequest,
  signal?: AbortSignal
): Promise<TelegramReportExportResponse> {
  return apiJson<TelegramReportExportResponse>('/api/v1/reports/telegram/export', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
