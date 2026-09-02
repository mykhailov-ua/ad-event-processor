import type { FlowPath } from '@/api/types';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';

export type CampaignReportRow = {
  id: string;
  name: string;
  clicks: number;
  lpCtrPct: number;
  crPct: number;
  lpClicks: number;
  leads: number;
  epcUsd: number;
  cpcUsd: number;
  revenueUsd: number;
  costUsd: number;
  profitUsd: number;
  roiPct: number;
};

export type CampaignReportDimension =
  | 'paths'
  | 'offers'
  | 'landers'
  | 'rules'
  | 'tokens'
  | 'connection'
  | 'device'
  | 'country'
  | 'default';

function microToUsd(micro?: number): number {
  if (micro == null || !Number.isFinite(micro)) {
    return 0;
  }
  return micro / 1_000_000;
}

function readNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === 'string') {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function rowFromMetrics(
  id: string,
  name: string,
  metrics: {
    clicks?: number;
    conversions?: number;
    spend_micro?: number;
    revenue_micro?: number;
    profit_micro?: number;
    roi_pct?: number;
    cr_pct?: number;
    epc_micro?: number;
    cpc_micro?: number;
  },
): CampaignReportRow {
  const clicks = metrics.clicks ?? 0;
  const leads = metrics.conversions ?? 0;
  const costUsd = microToUsd(metrics.spend_micro);
  const revenueUsd = microToUsd(metrics.revenue_micro);
  const profitUsd =
    metrics.profit_micro != null
      ? microToUsd(metrics.profit_micro)
      : revenueUsd - costUsd;
  const crPct = metrics.cr_pct ?? (clicks > 0 ? (leads / clicks) * 100 : 0);
  const roiPct =
    metrics.roi_pct ??
    (costUsd > 0 ? (profitUsd / costUsd) * 100 : 0);
  const epcUsd =
    metrics.epc_micro != null
      ? microToUsd(metrics.epc_micro)
      : clicks > 0
        ? revenueUsd / clicks
        : 0;
  const cpcUsd =
    metrics.cpc_micro != null
      ? microToUsd(metrics.cpc_micro)
      : clicks > 0
        ? costUsd / clicks
        : 0;

  return {
    id,
    name,
    clicks,
    lpCtrPct: 0,
    crPct,
    lpClicks: clicks,
    leads,
    epcUsd,
    cpcUsd,
    revenueUsd,
    costUsd,
    profitUsd,
    roiPct,
  };
}

export function buildCampaignReportRows({
  payload,
  dimension,
  flowPaths,
}: {
  payload: Record<string, unknown> | undefined;
  dimension: CampaignReportDimension;
  flowPaths?: FlowPath[];
}): CampaignReportRow[] {
  if (!payload) {
    return [];
  }

  const kpis = payload.kpis;
  const kpiRecord =
    kpis != null && typeof kpis === 'object' && !Array.isArray(kpis)
      ? (kpis as Record<string, unknown>)
      : undefined;

  if (dimension === 'paths' && flowPaths?.length) {
    return flowPaths.map((path, index) =>
      rowFromMetrics(`path-${index}`, `Path ${index + 1}`, {
        clicks: 0,
        conversions: 0,
        spend_micro: 0,
        revenue_micro: 0,
        profit_micro: 0,
        roi_pct: 0,
      }),
    );
  }

  if (dimension === 'offers' && flowPaths?.length) {
    const rows: CampaignReportRow[] = [];
    for (const [pathIndex, path] of flowPaths.entries()) {
      for (const [offerIndex, offer] of (path.offers ?? []).entries()) {
        rows.push(
          rowFromMetrics(
            `offer-${pathIndex}-${offerIndex}`,
            offer.offer_id?.slice(0, 8) ?? `Offer ${offerIndex + 1}`,
            { clicks: 0, conversions: 0 },
          ),
        );
      }
    }
    if (rows.length > 0) {
      return rows;
    }
  }

  const series = payload.series;
  if (Array.isArray(series) && series.length > 0) {
    return series.map((point, index) => {
      const record =
        point != null && typeof point === 'object' && !Array.isArray(point)
          ? (point as Record<string, unknown>)
          : {};
      const label = typeof record.label === 'string' ? record.label : `Row ${index + 1}`;
      return rowFromMetrics(`series-${index}`, label, {
        clicks: readNumber(record.clicks),
        conversions: readNumber(record.conversions),
        spend_micro: readNumber(record.spend_micro ?? record.spend_micros),
        revenue_micro: readNumber(record.revenue_micro),
        profit_micro: readNumber(record.profit_micro),
        roi_pct: readNumber(record.roi_pct),
      });
    });
  }

  if (kpiRecord) {
    return [
      rowFromMetrics('campaign-total', 'Campaign total', {
        clicks: readNumber(kpiRecord.clicks),
        conversions: readNumber(kpiRecord.conversions),
        spend_micro: readNumber(kpiRecord.spend_micro),
        revenue_micro: readNumber(kpiRecord.revenue_micro),
        profit_micro: readNumber(kpiRecord.profit_micro),
        roi_pct: readNumber(kpiRecord.roi_pct),
        cr_pct: readNumber(kpiRecord.cr_pct),
        epc_micro: readNumber(kpiRecord.epc_micro),
        cpc_micro: readNumber(kpiRecord.cpc_micro),
      }),
    ];
  }

  return [];
}

export function formatReportMoneyUsd(value: number): string {
  if (!Number.isFinite(value) || value === 0) {
    return '0.00 $';
  }
  return `${formatDashboardUsdFromMicro(Math.round(value * 1_000_000)).replace('$', '')} $`;
}

export function formatReportPct(value: number): string {
  if (!Number.isFinite(value) || value === 0) {
    return '0.00%';
  }
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}
