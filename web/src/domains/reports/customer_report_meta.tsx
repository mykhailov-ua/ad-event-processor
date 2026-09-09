import type { ReactNode } from 'react';

import {
  getCampaignGeoDeviceReport,
  getCampaignOverviewReport,
  getConversionTypePayoutReport,
  getCostSyncCoverageReport,
  getCustomerPortfolioReport,
  getDataQualityReport,
  getDaypartHeatmapReport,
  getDiscrepancyBuySellReport,
  getGeoRoiReport,
  getKeywordsReport,
  getPacingDriftReport,
  getPlacementsReport,
  getSpendVelocityReport,
  getTrafficSourcesReport,
  getTrueRoiReport,
} from '@/api/reports_api';
import type {
  CampaignGeoDeviceRow,
  CampaignOverviewRow,
  ConversionTypePayoutRow,
  CostSyncCoverageRow,
  CustomerPortfolioCampaignRow,
  CustomerPortfolioSummary,
  DataQualityRow,
  DaypartHeatmapRow,
  DiscrepancyBuySellRow,
  GeoROIRow,
  KeywordReportRow,
  PacingDriftRow,
  PlacementReportRow,
  SpendVelocityRow,
  TrafficSourceRow,
  TrueRoiReportRow,
} from '@/api/types';
import { formatPct, formatRatio, metricCell } from '@/domains/reports/report_metric_display';
import type { CustomerReportFetcher } from '@/domains/reports/use_customer_scoped_report_workspace';
import { displayCount, displayMicro } from '@/lib/display';
import { adminTypography } from '@/lib/admin_spacing';
import { resolveEconomicsProfitMicro, resolveEconomicsRoiPct } from '@/lib/economics';
import { Badge } from '@/components/ui/badge';

export type CustomerReportKey =
  | 'placements'
  | 'keywords'
  | 'geo-roi'
  | 'traffic-sources'
  | 'data-quality'
  | 'pacing-drift'
  | 'conversion-type-payout'
  | 'campaign-overview'
  | 'campaign-geo-device'
  | 'spend-velocity'
  | 'daypart-heatmap'
  | 'true-roi'
  | 'cost-sync-coverage'
  | 'customer-portfolio'
  | 'discrepancy-buy-sell';

export type CustomerReportColumn<Row> = {
  id: string;
  label: string;
  align?: 'end';
  cell: (row: Row) => ReactNode;
};

export type CustomerReportConfig<Row> = {
  key: CustomerReportKey;
  title: string;
  description: string;
  showCompare: boolean;
  fetch: CustomerReportFetcher<Row>;
  columns: CustomerReportColumn<Row>[];
  rowKey: (row: Row, index: number) => string;
  summaryBand?: (extras?: Record<string, unknown>) => ReactNode;
  badge?: (data: { freshness?: { stale?: boolean; as_of?: string } }) => ReactNode;
};

const metricsColumns = {
  impressions: (compare?: { impressions_delta?: number }) => ({
    id: 'impressions',
    label: 'Impressions',
    align: 'end' as const,
    cell: (row: { impressions?: number; compare?: { impressions_delta?: number } }) =>
      metricCell(row.impressions, row.compare?.impressions_delta ?? compare?.impressions_delta),
  }),
  clicks: {
    id: 'clicks',
    label: 'Clicks',
    align: 'end' as const,
    cell: (row: { clicks?: number; compare?: { clicks_delta?: number } }) =>
      metricCell(row.clicks, row.compare?.clicks_delta),
  },
  conversions: {
    id: 'conversions',
    label: 'Conversions',
    align: 'end' as const,
    cell: (row: { conversions?: number; compare?: { conversions_delta?: number } }) =>
      metricCell(row.conversions, row.compare?.conversions_delta),
  },
  spend: {
    id: 'spend',
    label: 'Spend',
    align: 'end' as const,
    cell: (row: { spend_micro?: number; compare?: { spend_micro_delta?: number } }) =>
      metricCell(row.spend_micro, row.compare?.spend_micro_delta, 'micro'),
  },
  revenue: {
    id: 'revenue',
    label: 'Revenue',
    align: 'end' as const,
    cell: (row: { revenue_micro?: number; compare?: { revenue_micro_delta?: number } }) =>
      metricCell(row.revenue_micro, row.compare?.revenue_micro_delta, 'micro'),
  },
  profit: {
    id: 'profit',
    label: 'Profit',
    align: 'end' as const,
    cell: (row: { revenue_micro?: number; spend_micro?: number; profit_micro?: number }) =>
      displayMicro(resolveEconomicsProfitMicro(row)) || '-',
  },
  roi: {
    id: 'roi',
    label: 'ROI',
    align: 'end' as const,
    cell: (row: {
      revenue_micro?: number;
      spend_micro?: number;
      profit_micro?: number;
      roi_pct?: number;
    }) => formatPct(resolveEconomicsRoiPct(row)),
  },
  cpa: {
    id: 'cpa',
    label: 'CPA',
    align: 'end' as const,
    cell: (row: { cpa_micro?: number }) => displayMicro(row.cpa_micro) || '-',
  },
  ctr: {
    id: 'ctr',
    label: 'CTR',
    align: 'end' as const,
    cell: (row: { ctr?: number }) => formatRatio(row.ctr),
  },
  ivt: {
    id: 'ivt',
    label: 'IVT',
    align: 'end' as const,
    cell: (row: { ivt_rate?: number }) => formatRatio(row.ivt_rate),
  },
};

async function fetchPlacements(
  params: Parameters<typeof getPlacementsReport>[0],
  signal?: AbortSignal
) {
  const payload = await getPlacementsReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchKeywords(
  params: Parameters<typeof getKeywordsReport>[0],
  signal?: AbortSignal
) {
  const payload = await getKeywordsReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchGeoRoi(params: Parameters<typeof getGeoRoiReport>[0], signal?: AbortSignal) {
  const payload = await getGeoRoiReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchTrafficSources(
  params: Parameters<typeof getTrafficSourcesReport>[0],
  signal?: AbortSignal
) {
  const payload = await getTrafficSourcesReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchDataQuality(
  params: Parameters<typeof getDataQualityReport>[0],
  signal?: AbortSignal
) {
  const payload = await getDataQualityReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
    extras: {
      telemetry_missing_rate_display: payload.telemetry_missing_rate_display,
      ch_lag_seconds_by_report_key: payload.ch_lag_seconds_by_report_key,
    },
  };
}

async function fetchPacingDrift(
  params: Parameters<typeof getPacingDriftReport>[0],
  signal?: AbortSignal
) {
  const payload = await getPacingDriftReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchConversionTypePayout(
  params: Parameters<typeof getConversionTypePayoutReport>[0],
  signal?: AbortSignal
) {
  const payload = await getConversionTypePayoutReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchCampaignOverview(
  params: Parameters<typeof getCampaignOverviewReport>[0],
  signal?: AbortSignal
) {
  const payload = await getCampaignOverviewReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness };
}

async function fetchCampaignGeoDevice(
  params: Parameters<typeof getCampaignGeoDeviceReport>[0],
  signal?: AbortSignal
) {
  const payload = await getCampaignGeoDeviceReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchSpendVelocity(
  params: Parameters<typeof getSpendVelocityReport>[0],
  signal?: AbortSignal
) {
  const payload = await getSpendVelocityReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchDaypartHeatmap(
  params: Parameters<typeof getDaypartHeatmapReport>[0],
  signal?: AbortSignal
) {
  const payload = await getDaypartHeatmapReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchTrueRoi(params: Parameters<typeof getTrueRoiReport>[0], signal?: AbortSignal) {
  const payload = await getTrueRoiReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchCostSyncCoverage(
  params: Parameters<typeof getCostSyncCoverageReport>[0],
  signal?: AbortSignal
) {
  const payload = await getCostSyncCoverageReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchDiscrepancyBuySell(
  params: Parameters<typeof getDiscrepancyBuySellReport>[0],
  signal?: AbortSignal
) {
  const payload = await getDiscrepancyBuySellReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchCustomerPortfolio(
  params: Parameters<typeof getCustomerPortfolioReport>[0],
  signal?: AbortSignal
) {
  const payload = await getCustomerPortfolioReport(params, signal);
  return {
    rows: payload.campaigns ?? [],
    freshness: payload.freshness,
    extras: { summary: payload.summary },
  };
}

function portfolioSummaryBand(extras?: Record<string, unknown>) {
  const summary = extras?.summary as CustomerPortfolioSummary | undefined;
  if (!summary) {
    return null;
  }
  const items = [
    { label: 'Active', value: displayCount(summary.active) },
    { label: 'Paused', value: displayCount(summary.paused) },
    { label: 'Archived', value: displayCount(summary.archived) },
    { label: 'Impressions (7d)', value: displayCount(summary.impressions_7d) },
    { label: 'Clicks (7d)', value: displayCount(summary.clicks_7d) },
    { label: 'Overspend', value: displayCount(summary.overspend_count) },
    { label: 'Attention', value: displayCount(summary.attention_count) },
    { label: 'Campaigns', value: displayCount(summary.campaigns_sample) },
  ];
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {items.map((item) => (
        <div key={item.label} className="rounded-md border p-3">
          <div className={adminTypography.captionPlain}>{item.label}</div>
          <div className="text-lg font-medium">{item.value || '-'}</div>
        </div>
      ))}
    </div>
  );
}

function formatDaypartHour(hour?: number): string {
  if (hour == null) {
    return '-';
  }
  return `${String(hour).padStart(2, '0')}:00`;
}

export const CUSTOMER_REPORT_CONFIGS: {
  placements: CustomerReportConfig<PlacementReportRow>;
  keywords: CustomerReportConfig<KeywordReportRow>;
  'geo-roi': CustomerReportConfig<GeoROIRow>;
  'traffic-sources': CustomerReportConfig<TrafficSourceRow>;
  'data-quality': CustomerReportConfig<DataQualityRow>;
  'pacing-drift': CustomerReportConfig<PacingDriftRow>;
  'conversion-type-payout': CustomerReportConfig<ConversionTypePayoutRow>;
  'campaign-overview': CustomerReportConfig<CampaignOverviewRow>;
  'campaign-geo-device': CustomerReportConfig<CampaignGeoDeviceRow>;
  'spend-velocity': CustomerReportConfig<SpendVelocityRow>;
  'daypart-heatmap': CustomerReportConfig<DaypartHeatmapRow>;
  'true-roi': CustomerReportConfig<TrueRoiReportRow>;
  'cost-sync-coverage': CustomerReportConfig<CostSyncCoverageRow>;
  'customer-portfolio': CustomerReportConfig<CustomerPortfolioCampaignRow>;
  'discrepancy-buy-sell': CustomerReportConfig<DiscrepancyBuySellRow>;
} = {
  placements: {
    key: 'placements',
    title: 'Placements',
    description: 'Placement performance by campaign.',
    showCompare: true,
    fetch: fetchPlacements,
    rowKey: (row, index) => `${row.placement_id ?? 'p'}-${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      {
        id: 'placement',
        label: 'Placement',
        cell: (row) => <span className={adminTypography.monoData}>{row.placement_id ?? '-'}</span>,
      },
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      metricsColumns.impressions(),
      metricsColumns.clicks,
      metricsColumns.conversions,
      metricsColumns.spend,
      metricsColumns.revenue,
      metricsColumns.profit,
      metricsColumns.roi,
      metricsColumns.cpa,
      metricsColumns.ctr,
      metricsColumns.ivt,
    ],
  },
  keywords: {
    key: 'keywords',
    title: 'Keywords',
    description: 'Keyword performance by campaign.',
    showCompare: true,
    fetch: fetchKeywords,
    rowKey: (row, index) => `${row.keyword ?? 'k'}-${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      { id: 'keyword', label: 'Keyword', cell: (row) => row.keyword ?? '-' },
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      metricsColumns.impressions(),
      metricsColumns.clicks,
      metricsColumns.conversions,
      metricsColumns.spend,
      metricsColumns.revenue,
      metricsColumns.profit,
      metricsColumns.roi,
      metricsColumns.cpa,
      metricsColumns.ctr,
      metricsColumns.ivt,
    ],
  },
  'geo-roi': {
    key: 'geo-roi',
    title: 'Geo ROI',
    description: 'ROI and funnel metrics by country.',
    showCompare: true,
    fetch: fetchGeoRoi,
    rowKey: (row, index) => `${row.country ?? 'ZZ'}-${index}`,
    columns: [
      { id: 'country', label: 'Country', cell: (row) => row.country ?? '-' },
      metricsColumns.impressions(),
      metricsColumns.clicks,
      metricsColumns.conversions,
      {
        id: 'ivt_events',
        label: 'IVT events',
        align: 'end',
        cell: (row) => displayCount(row.ivt_events) || '-',
      },
      metricsColumns.ivt,
      metricsColumns.spend,
      metricsColumns.revenue,
      metricsColumns.profit,
      metricsColumns.roi,
      metricsColumns.ctr,
    ],
  },
  'traffic-sources': {
    key: 'traffic-sources',
    title: 'Traffic sources',
    description: 'Channel rollup across campaigns.',
    showCompare: true,
    fetch: fetchTrafficSources,
    rowKey: (row, index) => `${row.channel ?? 'ch'}-${index}`,
    columns: [
      { id: 'channel', label: 'Channel', cell: (row) => row.channel ?? '-' },
      metricsColumns.impressions(),
      metricsColumns.clicks,
      metricsColumns.conversions,
      metricsColumns.spend,
      metricsColumns.revenue,
      metricsColumns.profit,
      metricsColumns.roi,
      metricsColumns.ctr,
    ],
  },
  'data-quality': {
    key: 'data-quality',
    title: 'Data quality',
    description: 'Postgres vs ClickHouse daily totals by campaign.',
    showCompare: false,
    fetch: fetchDataQuality,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${row.date ?? 'd'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      { id: 'date', label: 'Date', cell: (row) => row.date ?? '-' },
      {
        id: 'postgres',
        label: 'Postgres',
        align: 'end',
        cell: (row) => displayCount(row.postgres_total) || '-',
      },
      {
        id: 'clickhouse',
        label: 'ClickHouse',
        align: 'end',
        cell: (row) => displayCount(row.clickhouse_total) || '-',
      },
      {
        id: 'diff',
        label: 'Diff %',
        align: 'end',
        cell: (row) => formatRatio(row.diff_pct),
      },
      {
        id: 'severity',
        label: 'Severity',
        cell: (row) => (row.severity ? <Badge variant="outline">{row.severity}</Badge> : '-'),
      },
    ],
  },
  'pacing-drift': {
    key: 'pacing-drift',
    title: 'Pacing drift',
    description: 'Planned vs actual spend by campaign and day.',
    showCompare: false,
    fetch: fetchPacingDrift,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${row.date ?? 'd'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      { id: 'date', label: 'Date', cell: (row) => row.date ?? '-' },
      {
        id: 'planned',
        label: 'Planned',
        align: 'end',
        cell: (row) => displayMicro(row.planned_spend_micro) || '-',
      },
      {
        id: 'actual',
        label: 'Actual',
        align: 'end',
        cell: (row) => displayMicro(row.actual_spend_micro) || '-',
      },
      {
        id: 'drift',
        label: 'Drift',
        align: 'end',
        cell: (row) => formatRatio(row.drift_pct),
      },
      { id: 'mode', label: 'Pacing', cell: (row) => row.pacing_mode ?? '-' },
    ],
  },
  'conversion-type-payout': {
    key: 'conversion-type-payout',
    title: 'Conversion type payout',
    description: 'Payout rollup grouped by conversion goal name.',
    showCompare: false,
    fetch: fetchConversionTypePayout,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${row.goal_name ?? 'g'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      { id: 'goal', label: 'Goal', cell: (row) => row.goal_name ?? '-' },
      {
        id: 'conversions',
        label: 'Conversions',
        align: 'end',
        cell: (row) => displayCount(row.conversions) || '-',
      },
      {
        id: 'payout',
        label: 'Payout',
        align: 'end',
        cell: (row) => displayMicro(row.payout_micro) || '-',
      },
    ],
  },
  'campaign-overview': {
    key: 'campaign-overview',
    title: 'Campaign overview',
    description: 'Campaign economics and pacing signals (7-day rollup).',
    showCompare: false,
    fetch: fetchCampaignOverview,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      { id: 'name', label: 'Name', cell: (row) => row.name ?? '-' },
      { id: 'status', label: 'Status', cell: (row) => row.status ?? '-' },
      {
        id: 'impressions',
        label: 'Impressions (7d)',
        align: 'end',
        cell: (row) => displayCount(row.impressions_7d) || '-',
      },
      {
        id: 'clicks',
        label: 'Clicks (7d)',
        align: 'end',
        cell: (row) => displayCount(row.clicks_7d) || '-',
      },
      {
        id: 'utilization',
        label: 'Utilization',
        align: 'end',
        cell: (row) => formatRatio(row.utilization_pct),
      },
      {
        id: 'pacing_drift',
        label: 'Pacing drift',
        align: 'end',
        cell: (row) => formatRatio(row.pacing_drift_pct),
      },
      {
        id: 'overspend',
        label: 'Overspend risk',
        cell: (row) => (row.overspend_risk ? <Badge variant="destructive">Risk</Badge> : '-'),
      },
    ],
  },
  'campaign-geo-device': {
    key: 'campaign-geo-device',
    title: 'Campaign geo and device',
    description: 'Click volume by country and device.',
    showCompare: false,
    fetch: fetchCampaignGeoDevice,
    rowKey: (row, index) => `${row.country ?? 'ZZ'}-${row.device ?? 'd'}-${index}`,
    columns: [
      { id: 'country', label: 'Country', cell: (row) => row.country ?? '-' },
      { id: 'device', label: 'Device', cell: (row) => row.device ?? '-' },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => displayCount(row.clicks) || '-',
      },
    ],
  },
  'spend-velocity': {
    key: 'spend-velocity',
    title: 'Spend velocity',
    description: 'Hourly spend and click volume.',
    showCompare: true,
    fetch: fetchSpendVelocity,
    rowKey: (row, index) => `${row.bucket ?? 'b'}-${index}`,
    columns: [
      { id: 'bucket', label: 'Hour', cell: (row) => row.bucket ?? '-' },
      {
        id: 'spend',
        label: 'Spend',
        align: 'end',
        cell: (row) => metricCell(row.spend_micro, row.compare?.spend_micro_delta, 'micro'),
      },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => metricCell(row.clicks, row.compare?.clicks_delta),
      },
    ],
  },
  'daypart-heatmap': {
    key: 'daypart-heatmap',
    title: 'Daypart heatmap',
    description: 'Click volume by hour of day (UTC).',
    showCompare: true,
    fetch: fetchDaypartHeatmap,
    rowKey: (row, index) => `${row.hour ?? 'h'}-${index}`,
    columns: [
      { id: 'hour', label: 'Hour (UTC)', cell: (row) => formatDaypartHour(row.hour) },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => metricCell(row.clicks, row.compare?.clicks_delta),
      },
    ],
  },
  'true-roi': {
    key: 'true-roi',
    title: 'True ROI',
    description: 'ROI after synced ad spend and revenue.',
    showCompare: true,
    fetch: fetchTrueRoi,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      {
        id: 'ad_spend',
        label: 'Ad spend',
        align: 'end',
        cell: (row) => metricCell(row.ad_spend_micro, row.compare?.spend_micro_delta, 'micro'),
      },
      {
        id: 'revenue',
        label: 'Revenue',
        align: 'end',
        cell: (row) => metricCell(row.revenue_micro, row.compare?.revenue_micro_delta, 'micro'),
      },
      {
        id: 'profit',
        label: 'True profit',
        align: 'end',
        cell: (row) => displayMicro(resolveEconomicsProfitMicro(row, row.true_profit_micro)) || '-',
      },
      {
        id: 'roi',
        label: 'True ROI',
        align: 'end',
        cell: (row) => formatPct(resolveEconomicsRoiPct(row, row.true_roi_pct)),
      },
      {
        id: 'cpa',
        label: 'True CPA',
        align: 'end',
        cell: (row) => displayMicro(row.true_cpa_micro) || '-',
      },
      {
        id: 'conversions',
        label: 'Conversions',
        align: 'end',
        cell: (row) => metricCell(row.conversions, row.compare?.conversions_delta),
      },
    ],
  },
  'cost-sync-coverage': {
    key: 'cost-sync-coverage',
    title: 'Cost sync coverage',
    description: 'Campaigns with clicks but missing cost snapshots.',
    showCompare: false,
    fetch: fetchCostSyncCoverage,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => displayCount(row.clicks) || '-',
      },
      {
        id: 'spend',
        label: 'Spend',
        align: 'end',
        cell: (row) => displayMicro(row.spend_micro) || '-',
      },
      { id: 'gap', label: 'Coverage gap', cell: (row) => row.coverage_gap ?? '-' },
      { id: 'network', label: 'Network', cell: (row) => row.network ?? '-' },
      { id: 'sync', label: 'Last sync', cell: (row) => row.last_sync_status ?? '-' },
    ],
  },
  'customer-portfolio': {
    key: 'customer-portfolio',
    title: 'Customer portfolio',
    description: 'Portfolio KPIs and campaign pacing snapshot.',
    showCompare: false,
    fetch: fetchCustomerPortfolio,
    summaryBand: portfolioSummaryBand,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      { id: 'name', label: 'Name', cell: (row) => row.name ?? '-' },
      { id: 'status', label: 'Status', cell: (row) => row.status ?? '-' },
      {
        id: 'impressions',
        label: 'Impressions (7d)',
        align: 'end',
        cell: (row) => displayCount(row.impressions_7d) || '-',
      },
      {
        id: 'clicks',
        label: 'Clicks (7d)',
        align: 'end',
        cell: (row) => displayCount(row.clicks_7d) || '-',
      },
      {
        id: 'utilization',
        label: 'Utilization',
        align: 'end',
        cell: (row) => formatRatio(row.utilization_pct),
      },
      {
        id: 'pacing_drift',
        label: 'Pacing drift',
        align: 'end',
        cell: (row) => formatRatio(row.pacing_drift_pct),
      },
      {
        id: 'overspend',
        label: 'Overspend risk',
        cell: (row) => (row.overspend_risk ? <Badge variant="destructive">Risk</Badge> : '-'),
      },
    ],
  },
  'discrepancy-buy-sell': {
    key: 'discrepancy-buy-sell',
    title: 'Buy vs sell discrepancy',
    description: 'Buy spend vs sell revenue delta by campaign.',
    showCompare: false,
    fetch: fetchDiscrepancyBuySell,
    rowKey: (row, index) => `${row.campaign_id ?? 'c'}-${index}`,
    columns: [
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className={adminTypography.monoData}>{row.campaign_id ?? '-'}</span>,
      },
      {
        id: 'buy',
        label: 'Buy spend',
        align: 'end',
        cell: (row) => displayMicro(row.buy_spend_micro) || '-',
      },
      {
        id: 'sell',
        label: 'Sell revenue',
        align: 'end',
        cell: (row) => displayMicro(row.sell_rev_micro) || '-',
      },
      {
        id: 'delta',
        label: 'Delta',
        align: 'end',
        cell: (row) => displayMicro(row.delta_micro) || '-',
      },
      {
        id: 'delta_pct',
        label: 'Delta %',
        align: 'end',
        cell: (row) => formatRatio(row.delta_pct),
      },
    ],
  },
};
