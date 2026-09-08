import type { ReactNode } from 'react';

import {
  getMlFeatureSpikesReport,
  getMlScoreDistributionReport,
  getMlShadowDeltaReport,
} from '@/api/reports_api';
import type {
  MLFeatureSpikeRow,
  MLScoreBucketRow,
  MLShadowDeltaRow,
  DataFreshness,
  MlReportQuery,
} from '@/api/types';
import { formatRatio } from '@/domains/reports/report_metric_display';
import { displayCount, displayTimestamp } from '@/lib/display';

export type MlReportKey = 'ml/score-distribution' | 'ml/shadow-delta' | 'ml/feature-spikes';

export type MlReportFetcher<Row> = (
  params: MlReportQuery,
  signal?: AbortSignal
) => Promise<{
  rows: Row[];
  freshness?: DataFreshness;
  next_cursor?: string;
}>;

export type MlReportColumn<Row> = {
  id: string;
  label: string;
  align?: 'end';
  cell: (row: Row) => ReactNode;
};

export type MlReportConfig<Row> = {
  key: MlReportKey;
  title: string;
  description: string;
  fetch: MlReportFetcher<Row>;
  columns: MlReportColumn<Row>[];
  rowKey: (row: Row, index: number) => string;
};

async function fetchScoreDistribution(params: MlReportQuery, signal?: AbortSignal) {
  const payload = await getMlScoreDistributionReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchShadowDelta(params: MlReportQuery, signal?: AbortSignal) {
  const payload = await getMlShadowDeltaReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

async function fetchFeatureSpikes(params: MlReportQuery, signal?: AbortSignal) {
  const payload = await getMlFeatureSpikesReport(params, signal);
  return {
    rows: payload.rows ?? [],
    freshness: payload.freshness,
    next_cursor: payload.next_cursor,
  };
}

export const ML_REPORT_CONFIGS: {
  [K in MlReportKey]: MlReportConfig<
    K extends 'ml/score-distribution'
      ? MLScoreBucketRow
      : K extends 'ml/shadow-delta'
        ? MLShadowDeltaRow
        : MLFeatureSpikeRow
  >;
} = {
  'ml/score-distribution': {
    key: 'ml/score-distribution',
    title: 'ML score distribution',
    description: 'Histogram of shadow ML scores by bucket.',
    fetch: fetchScoreDistribution,
    rowKey: (row, index) => `${row.score_bucket ?? index}`,
    columns: [
      {
        id: 'score_bucket',
        label: 'Score bucket',
        align: 'end',
        cell: (row) => formatRatio(row.score_bucket),
      },
      {
        id: 'row_count',
        label: 'Rows',
        align: 'end',
        cell: (row) => displayCount(row.row_count) || '-',
      },
    ],
  },
  'ml/shadow-delta': {
    key: 'ml/shadow-delta',
    title: 'Shadow vs live ML delta',
    description: 'Hourly shadow score averages vs feature event density.',
    fetch: fetchShadowDelta,
    rowKey: (row, index) => row.bucket ?? String(index),
    columns: [
      {
        id: 'bucket',
        label: 'Bucket',
        cell: (row) => displayTimestamp(row.bucket) || '-',
      },
      {
        id: 'avg_shadow_score',
        label: 'Avg shadow score',
        align: 'end',
        cell: (row) => formatRatio(row.avg_shadow_score),
      },
      {
        id: 'score_count',
        label: 'Score count',
        align: 'end',
        cell: (row) => displayCount(row.score_count) || '-',
      },
      {
        id: 'avg_feature_events',
        label: 'Avg feature events',
        align: 'end',
        cell: (row) => displayCount(row.avg_feature_events) || '-',
      },
    ],
  },
  'ml/feature-spikes': {
    key: 'ml/feature-spikes',
    title: 'ML feature spikes',
    description: 'Minute windows with elevated feature event volume.',
    fetch: fetchFeatureSpikes,
    rowKey: (row, index) => row.window_start ?? String(index),
    columns: [
      {
        id: 'window_start',
        label: 'Window start',
        cell: (row) => displayTimestamp(row.window_start) || '-',
      },
      {
        id: 'events',
        label: 'Events',
        align: 'end',
        cell: (row) => displayCount(row.events) || '-',
      },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => displayCount(row.clicks) || '-',
      },
      {
        id: 'campaigns',
        label: 'Campaigns',
        align: 'end',
        cell: (row) => displayCount(row.campaigns) || '-',
      },
    ],
  },
};
