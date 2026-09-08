import type { ReactNode } from 'react';

import {
  getConversionTypePayoutReport,
  getDataQualityReport,
  getGeoRoiReport,
  getKeywordsReport,
  getPacingDriftReport,
  getPlacementsReport,
  getTrafficSourcesReport,
} from '@/api/reports_api';
import type {
  ConversionTypePayoutRow,
  DataQualityRow,
  GeoROIRow,
  KeywordReportRow,
  PacingDriftRow,
  PlacementReportRow,
  TrafficSourceRow,
} from '@/api/types';
import { formatPct, formatRatio, metricCell } from '@/domains/reports/report_metric_display';
import type { CustomerReportFetcher } from '@/domains/reports/use_customer_scoped_report_workspace';
import { displayCount, displayMicro } from '@/lib/display';
import { Badge } from '@/components/ui/badge';

export type CustomerReportKey =
  | 'placements'
  | 'keywords'
  | 'geo-roi'
  | 'traffic-sources'
  | 'data-quality'
  | 'pacing-drift'
  | 'conversion-type-payout';

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
    cell: (row: { profit_micro?: number }) => displayMicro(row.profit_micro) || '-',
  },
  roi: {
    id: 'roi',
    label: 'ROI',
    align: 'end' as const,
    cell: (row: { roi_pct?: number }) => formatPct(row.roi_pct),
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
  return { rows: payload.rows ?? [], freshness: payload.freshness, next_cursor: payload.next_cursor };
}

async function fetchKeywords(
  params: Parameters<typeof getKeywordsReport>[0],
  signal?: AbortSignal
) {
  const payload = await getKeywordsReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness, next_cursor: payload.next_cursor };
}

async function fetchGeoRoi(
  params: Parameters<typeof getGeoRoiReport>[0],
  signal?: AbortSignal
) {
  const payload = await getGeoRoiReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness, next_cursor: payload.next_cursor };
}

async function fetchTrafficSources(
  params: Parameters<typeof getTrafficSourcesReport>[0],
  signal?: AbortSignal
) {
  const payload = await getTrafficSourcesReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness, next_cursor: payload.next_cursor };
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
  return { rows: payload.rows ?? [], freshness: payload.freshness, next_cursor: payload.next_cursor };
}

async function fetchConversionTypePayout(
  params: Parameters<typeof getConversionTypePayoutReport>[0],
  signal?: AbortSignal
) {
  const payload = await getConversionTypePayoutReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness, next_cursor: payload.next_cursor };
}

export const CUSTOMER_REPORT_CONFIGS: {
  placements: CustomerReportConfig<PlacementReportRow>;
  keywords: CustomerReportConfig<KeywordReportRow>;
  'geo-roi': CustomerReportConfig<GeoROIRow>;
  'traffic-sources': CustomerReportConfig<TrafficSourceRow>;
  'data-quality': CustomerReportConfig<DataQualityRow>;
  'pacing-drift': CustomerReportConfig<PacingDriftRow>;
  'conversion-type-payout': CustomerReportConfig<ConversionTypePayoutRow>;
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
        cell: (row) => <span className="font-mono text-xs">{row.placement_id ?? '-'}</span>,
      },
      {
        id: 'campaign',
        label: 'Campaign',
        cell: (row) => <span className="font-mono text-xs">{row.campaign_id ?? '-'}</span>,
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
        cell: (row) => <span className="font-mono text-xs">{row.campaign_id ?? '-'}</span>,
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
        cell: (row) => <span className="font-mono text-xs">{row.campaign_id ?? '-'}</span>,
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
        cell: (row) =>
          row.severity ? <Badge variant="outline">{row.severity}</Badge> : '-',
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
        cell: (row) => <span className="font-mono text-xs">{row.campaign_id ?? '-'}</span>,
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
        cell: (row) => <span className="font-mono text-xs">{row.campaign_id ?? '-'}</span>,
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
};
