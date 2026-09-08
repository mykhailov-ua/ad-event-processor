import type { ReactNode } from 'react';

import {
  getTelegramBotsReport,
  getTelegramFraudReport,
  getTelegramFunnelReport,
  getTelegramPremiumReport,
  getTelegramRollupReport,
  getTelegramSummaryReport,
} from '@/api/reports_api';
import type {
  TelegramBotBreakdownRow,
  TelegramFraudReportResponse,
  TelegramFunnelRow,
  TelegramPremiumReportResponse,
  TelegramReportFreshness,
  TelegramReportQuery,
  TelegramSummaryReportResponse,
} from '@/api/types';
import { ReportKpiGrid } from '@/domains/reports/report_kpi_grid';
import { formatPct } from '@/domains/reports/report_metric_display';
import { displayCount } from '@/lib/display';

export type TelegramReportKey =
  | 'telegram'
  | 'telegram/summary'
  | 'telegram/funnel'
  | 'telegram/bots'
  | 'telegram/premium'
  | 'telegram/fraud';

export type TelegramReportFetchResult<Row> = {
  rows: Row[];
  freshness?: TelegramReportFreshness;
  extras?: Record<string, unknown>;
};

export type TelegramReportFetcher<Row> = (
  params: TelegramReportQuery,
  signal?: AbortSignal
) => Promise<TelegramReportFetchResult<Row>>;

export type TelegramReportColumn<Row> = {
  id: string;
  label: string;
  align?: 'end';
  cell: (row: Row) => ReactNode;
};

export type TelegramReportConfig<Row> = {
  key: TelegramReportKey;
  title: string;
  description: string;
  enableExport?: boolean;
  fetch: TelegramReportFetcher<Row>;
  columns?: TelegramReportColumn<Row>[];
  rowKey?: (row: Row, index: number) => string;
  summaryBand?: (extras?: Record<string, unknown>) => ReactNode;
};

function summaryKpiBand(extras?: Record<string, unknown>) {
  const summary = extras?.summary as TelegramSummaryReportResponse | undefined;
  if (!summary) {
    return null;
  }
  return (
    <ReportKpiGrid
      items={[
        { label: 'Clicks', value: displayCount(summary.clicks) || '-' },
        { label: 'Impressions', value: displayCount(summary.impressions) || '-' },
        { label: 'Premium', value: displayCount(summary.premium) || '-' },
        { label: 'Motivated', value: displayCount(summary.motivated) || '-' },
        { label: 'Conversions', value: displayCount(summary.conversions) || '-' },
        {
          label: 'Funnel clicks',
          value: displayCount(summary.funnel?.clicks) || '-',
        },
      ]}
    />
  );
}

function premiumKpiBand(extras?: Record<string, unknown>) {
  const payload = extras?.premium as TelegramPremiumReportResponse | undefined;
  if (!payload) {
    return null;
  }
  return (
    <ReportKpiGrid
      items={[
        { label: 'Premium clicks', value: displayCount(payload.premium_clicks) || '-' },
        {
          label: 'Non-premium clicks',
          value: displayCount(payload.non_premium_clicks) || '-',
        },
        { label: 'Premium rate', value: formatPct(payload.premium_rate_pct) },
      ]}
    />
  );
}

function fraudKpiBand(extras?: Record<string, unknown>) {
  const payload = extras?.fraud as TelegramFraudReportResponse | undefined;
  if (!payload) {
    return null;
  }
  return (
    <ReportKpiGrid
      items={[
        { label: 'Blocked clicks', value: displayCount(payload.blocked_clicks) || '-' },
        { label: 'Shadow clicks', value: displayCount(payload.shadow_clicks) || '-' },
      ]}
    />
  );
}

async function fetchTelegramRollup(params: TelegramReportQuery, signal?: AbortSignal) {
  const payload = await getTelegramRollupReport(params, signal);
  return { rows: [], freshness: payload.freshness, extras: { summary: payload } };
}

async function fetchTelegramSummary(params: TelegramReportQuery, signal?: AbortSignal) {
  const payload = await getTelegramSummaryReport(params, signal);
  return { rows: [], freshness: payload.freshness, extras: { summary: payload } };
}

async function fetchTelegramFunnel(params: TelegramReportQuery, signal?: AbortSignal) {
  const payload = await getTelegramFunnelReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness };
}

async function fetchTelegramBots(params: TelegramReportQuery, signal?: AbortSignal) {
  const payload = await getTelegramBotsReport(params, signal);
  return { rows: payload.rows ?? [], freshness: payload.freshness };
}

async function fetchTelegramPremium(params: TelegramReportQuery, signal?: AbortSignal) {
  const payload = await getTelegramPremiumReport(params, signal);
  return { rows: [], freshness: payload.freshness, extras: { premium: payload } };
}

async function fetchTelegramFraud(params: TelegramReportQuery, signal?: AbortSignal) {
  const payload = await getTelegramFraudReport(params, signal);
  return { rows: [], freshness: payload.freshness, extras: { fraud: payload } };
}

export type TelegramKpiOnlyRow = Record<string, never>;

export const TELEGRAM_REPORT_CONFIGS: {
  [K in TelegramReportKey]: TelegramReportConfig<
    K extends 'telegram/funnel'
      ? TelegramFunnelRow
      : K extends 'telegram/bots'
        ? TelegramBotBreakdownRow
        : TelegramKpiOnlyRow
  >;
} = {
  telegram: {
    key: 'telegram',
    title: 'Telegram Mini App rollup',
    description: 'Telegram Mini App traffic and conversion KPIs.',
    enableExport: true,
    fetch: fetchTelegramRollup,
    summaryBand: summaryKpiBand,
  },
  'telegram/summary': {
    key: 'telegram/summary',
    title: 'Telegram summary KPIs',
    description: 'Telegram summary KPIs for the selected window.',
    fetch: fetchTelegramSummary,
    summaryBand: summaryKpiBand,
  },
  'telegram/funnel': {
    key: 'telegram/funnel',
    title: 'Telegram conversion funnel',
    description: 'Funnel metrics grouped by start_param.',
    fetch: fetchTelegramFunnel,
    rowKey: (row, index) => `${row.start_param ?? 'sp'}-${index}`,
    columns: [
      { id: 'start_param', label: 'Start param', cell: (row) => row.start_param ?? '-' },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => displayCount(row.clicks) || '-',
      },
      {
        id: 'impressions',
        label: 'Impressions',
        align: 'end',
        cell: (row) => displayCount(row.impressions) || '-',
      },
      {
        id: 'conversions',
        label: 'Conversions',
        align: 'end',
        cell: (row) => displayCount(row.conversions) || '-',
      },
    ],
  },
  'telegram/bots': {
    key: 'telegram/bots',
    title: 'Telegram bot performance',
    description: 'Click and impression volume by bot_id.',
    fetch: fetchTelegramBots,
    rowKey: (row, index) => `${row.bot_id ?? 'b'}-${index}`,
    columns: [
      {
        id: 'bot_id',
        label: 'Bot ID',
        cell: (row) => <span className="text-xs">{row.bot_id ?? '-'}</span>,
      },
      {
        id: 'clicks',
        label: 'Clicks',
        align: 'end',
        cell: (row) => displayCount(row.clicks) || '-',
      },
      {
        id: 'impressions',
        label: 'Impressions',
        align: 'end',
        cell: (row) => displayCount(row.impressions) || '-',
      },
      {
        id: 'premium',
        label: 'Premium',
        align: 'end',
        cell: (row) => displayCount(row.premium) || '-',
      },
    ],
  },
  'telegram/premium': {
    key: 'telegram/premium',
    title: 'Telegram premium users',
    description: 'Premium vs non-premium click split.',
    fetch: fetchTelegramPremium,
    summaryBand: premiumKpiBand,
  },
  'telegram/fraud': {
    key: 'telegram/fraud',
    title: 'Telegram fraud signals',
    description: 'Blocked and shadow Telegram click counts.',
    fetch: fetchTelegramFraud,
    summaryBand: fraudKpiBand,
  },
};
