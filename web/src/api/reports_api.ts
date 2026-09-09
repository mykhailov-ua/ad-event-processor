import { apiFetch, apiJson, apiJsonValidated, parseApiError } from './client.js';
import {
  parseFraudBreakdownReportResponse,
  parseFraudCatalogReportResponse,
  parseWireSignalBreakdownReportResponse,
} from './validate.js';
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
  PostbackReconReportResponse,
  PlacementReportResponse,
  KeywordReportResponse,
  GeoROIReportResponse,
  TrafficSourcesReportResponse,
  DataQualityReportResponse,
  PacingDriftReportResponse,
  ConversionTypePayoutReportResponse,
  CampaignOverviewReportResponse,
  CampaignGeoDeviceReportResponse,
  SpendVelocityReportResponse,
  DaypartHeatmapReportResponse,
  TrueRoiReportResponse,
  CostSyncCoverageReportResponse,
  DiscrepancyBuySellReportResponse,
  CustomerPortfolioReportResponse,
  EdgeParityReportResponse,
  MLFeatureSpikesReportResponse,
  MLScoreDistributionReportResponse,
  MLShadowDeltaReportResponse,
  MlReportQuery,
  TelegramSummaryReportResponse,
  TelegramFunnelReportResponse,
  TelegramBotsReportResponse,
  TelegramPremiumReportResponse,
  TelegramFraudReportResponse,
  TelegramReportQuery,
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
  FraudCatalogReportResponseMap,
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

export async function getFraudCatalogReport<K extends FraudCatalogReportKey>(
  key: K,
  params: FraudCatalogReportQuery = {},
  signal?: AbortSignal
): Promise<FraudCatalogReportResponseMap[K]> {
  return apiJsonValidated(
    buildFraudCatalogReportPath(key, params),
    { signal },
    (value) => parseFraudCatalogReportResponse(key, value)
  );
}

export function buildCampaignToggleCohortPath(params: CampaignToggleCohortQuery): string {
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
    const payload = await apiJsonValidated(path, { signal }, parseWireSignalBreakdownReportResponse);
    return {
      rows: payload.rows ?? [],
      freshness: payload.freshness,
      next_cursor: payload.next_cursor,
    };
  }
  const payload = await apiJsonValidated(path, { signal }, parseFraudBreakdownReportResponse);
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

export function buildCustomerReportPath(
  key: string,
  params: CustomerScopedReportQuery = {}
): string {
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
  return apiJson<PlacementReportResponse>(buildCustomerReportPath('placements', params), {
    signal,
  });
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
  return apiJson<TrafficSourcesReportResponse>(buildCustomerReportPath('traffic-sources', params), {
    signal,
  });
}

export async function getDataQualityReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<DataQualityReportResponse> {
  return apiJson<DataQualityReportResponse>(buildCustomerReportPath('data-quality', params), {
    signal,
  });
}

export async function getPacingDriftReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<PacingDriftReportResponse> {
  return apiJson<PacingDriftReportResponse>(buildCustomerReportPath('pacing-drift', params), {
    signal,
  });
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

export async function getCampaignOverviewReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<CampaignOverviewReportResponse> {
  return apiJson<CampaignOverviewReportResponse>(
    buildCustomerReportPath('campaign-overview', params),
    { signal }
  );
}

export async function getCampaignGeoDeviceReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<CampaignGeoDeviceReportResponse> {
  return apiJson<CampaignGeoDeviceReportResponse>(
    buildCustomerReportPath('campaign-geo-device', params),
    { signal }
  );
}

export async function getSpendVelocityReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<SpendVelocityReportResponse> {
  return apiJson<SpendVelocityReportResponse>(buildCustomerReportPath('spend-velocity', params), {
    signal,
  });
}

export async function getDaypartHeatmapReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<DaypartHeatmapReportResponse> {
  return apiJson<DaypartHeatmapReportResponse>(buildCustomerReportPath('daypart-heatmap', params), {
    signal,
  });
}

export async function getTrueRoiReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<TrueRoiReportResponse> {
  return apiJson<TrueRoiReportResponse>(buildCustomerReportPath('true-roi', params), { signal });
}

export async function getCostSyncCoverageReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<CostSyncCoverageReportResponse> {
  return apiJson<CostSyncCoverageReportResponse>(
    buildCustomerReportPath('cost-sync-coverage', params),
    { signal }
  );
}

export async function getDiscrepancyBuySellReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<DiscrepancyBuySellReportResponse> {
  return apiJson<DiscrepancyBuySellReportResponse>(
    buildCustomerReportPath('discrepancy-buy-sell', params),
    { signal }
  );
}

export async function getCustomerPortfolioReport(
  params: CustomerScopedReportQuery = {},
  signal?: AbortSignal
): Promise<CustomerPortfolioReportResponse> {
  return apiJson<CustomerPortfolioReportResponse>(
    buildCustomerReportPath('customer-portfolio', params),
    { signal }
  );
}

export function buildTelegramReportPath(key: string, params: TelegramReportQuery = {}): string {
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

  const query = search.toString();
  return query ? `${basePath}?${query}` : basePath;
}

export async function getEdgeParityReport(
  params: { from?: string; to?: string },
  signal?: AbortSignal
): Promise<EdgeParityReportResponse> {
  const search = new URLSearchParams();
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  const query = search.toString();
  const path = query ? `/api/v1/reports/edge-parity?${query}` : '/api/v1/reports/edge-parity';
  return apiJson<EdgeParityReportResponse>(path, { signal });
}

export function buildMlReportPath(key: string, params: MlReportQuery = {}): string {
  return buildReportRunPath(key, {
    from: params.from,
    to: params.to,
    limit: params.limit,
    offset: params.offset,
    cursor: params.cursor,
  });
}

export async function getMlScoreDistributionReport(
  params: MlReportQuery = {},
  signal?: AbortSignal
): Promise<MLScoreDistributionReportResponse> {
  return apiJson<MLScoreDistributionReportResponse>(
    buildMlReportPath('ml/score-distribution', params),
    { signal }
  );
}

export async function getMlShadowDeltaReport(
  params: MlReportQuery = {},
  signal?: AbortSignal
): Promise<MLShadowDeltaReportResponse> {
  return apiJson<MLShadowDeltaReportResponse>(buildMlReportPath('ml/shadow-delta', params), {
    signal,
  });
}

export async function getMlFeatureSpikesReport(
  params: MlReportQuery = {},
  signal?: AbortSignal
): Promise<MLFeatureSpikesReportResponse> {
  return apiJson<MLFeatureSpikesReportResponse>(buildMlReportPath('ml/feature-spikes', params), {
    signal,
  });
}

export async function getTelegramRollupReport(
  params: TelegramReportQuery = {},
  signal?: AbortSignal
): Promise<TelegramSummaryReportResponse> {
  return apiJson<TelegramSummaryReportResponse>(buildTelegramReportPath('telegram', params), {
    signal,
  });
}

export async function getTelegramSummaryReport(
  params: TelegramReportQuery = {},
  signal?: AbortSignal
): Promise<TelegramSummaryReportResponse> {
  return apiJson<TelegramSummaryReportResponse>(
    buildTelegramReportPath('telegram/summary', params),
    { signal }
  );
}

export async function getTelegramFunnelReport(
  params: TelegramReportQuery = {},
  signal?: AbortSignal
): Promise<TelegramFunnelReportResponse> {
  return apiJson<TelegramFunnelReportResponse>(buildTelegramReportPath('telegram/funnel', params), {
    signal,
  });
}

export async function getTelegramBotsReport(
  params: TelegramReportQuery = {},
  signal?: AbortSignal
): Promise<TelegramBotsReportResponse> {
  return apiJson<TelegramBotsReportResponse>(buildTelegramReportPath('telegram/bots', params), {
    signal,
  });
}

export async function getTelegramPremiumReport(
  params: TelegramReportQuery = {},
  signal?: AbortSignal
): Promise<TelegramPremiumReportResponse> {
  return apiJson<TelegramPremiumReportResponse>(
    buildTelegramReportPath('telegram/premium', params),
    { signal }
  );
}

export async function getTelegramFraudReport(
  params: TelegramReportQuery = {},
  signal?: AbortSignal
): Promise<TelegramFraudReportResponse> {
  return apiJson<TelegramFraudReportResponse>(buildTelegramReportPath('telegram/fraud', params), {
    signal,
  });
}

export async function getRtbOverviewReport(
  params: RtbReportQuery = {},
  signal?: AbortSignal
): Promise<RtbOverviewReportResponse> {
  return apiJson<RtbOverviewReportResponse>(buildReportRunPath('rtb-overview', params), { signal });
}

export async function getRtbNoBidReasonsReport(
  params: RtbReportQuery = {},
  signal?: AbortSignal
): Promise<RtbNoBidReasonsReportResponse> {
  return apiJson<RtbNoBidReasonsReportResponse>(buildReportRunPath('rtb-no-bid-reasons', params), {
    signal,
  });
}

export async function getRtbGeoDeviceReport(
  params: RtbReportQuery = {},
  signal?: AbortSignal
): Promise<RtbGeoDeviceReportResponse> {
  return apiJson<RtbGeoDeviceReportResponse>(buildReportRunPath('rtb-geo-device', params), {
    signal,
  });
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

export async function getCustomerFraudEvidenceReport(
  params: ReportRunQuery = {},
  signal?: AbortSignal
): Promise<FraudEvidencePack> {
  return runEvidencePackReport('customer-fraud-evidence', params, signal);
}

export async function getFraudEvidencePackReport(
  params: ReportRunQuery = {},
  signal?: AbortSignal
): Promise<FraudEvidencePack> {
  return runEvidencePackReport('fraud-evidence-pack', params, signal);
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
