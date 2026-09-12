import type { BuyerDashboardPayload } from '@/domains/dashboards/dashboard_types';
import { seedDeterministicUuid } from '@/lib/uuid';

type DashboardSeriesPoint = {
  label: string;
  impressions: number;
  clicks: number;
  conversions: number;
  spend_micro: number;
  revenue_micro: number;
  profit_micro: number;
};

export function isChartMockPreviewEnabled(search: string): boolean {
  const query = search.startsWith('?') ? search.slice(1) : search;
  return new URLSearchParams(query).get('chart_mock') === '1';
}

function parseIsoDay(iso: string): Date {
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) {
    return new Date();
  }
  return parsed;
}

function formatSeriesLabel(day: Date): string {
  const year = day.getUTCFullYear();
  const month = String(day.getUTCMonth() + 1).padStart(2, '0');
  const date = String(day.getUTCDate()).padStart(2, '0');
  return `${year}-${month}-${date}`;
}

function buildSyntheticSeries(from: string, to: string): DashboardSeriesPoint[] {
  const start = parseIsoDay(from);
  const end = parseIsoDay(to);
  if (end.getTime() < start.getTime()) {
    return [];
  }

  const points: DashboardSeriesPoint[] = [];
  const cursor = new Date(start);
  let dayIndex = 0;

  while (cursor.getTime() <= end.getTime() && points.length < 14) {
    const impressions = 18_437 + dayIndex * 1_283;
    const clicks = 2_917 + dayIndex * 173;
    const conversions = 127 + dayIndex * 11;
    const spendMicro = 4_827_350_000 + dayIndex * 318_470_000;
    const revenueMicro = spendMicro + 1_146_820_000 + dayIndex * 92_350_000;
    const profitMicro = revenueMicro - spendMicro;

    points.push({
      label: formatSeriesLabel(cursor),
      impressions,
      clicks,
      conversions,
      spend_micro: spendMicro,
      revenue_micro: revenueMicro,
      profit_micro: profitMicro,
    });

    cursor.setUTCDate(cursor.getUTCDate() + 1);
    dayIndex += 1;
  }

  return points;
}

export function buildBuyerDashboardChartMock(
  customerId: string,
  from: string,
  to: string
): BuyerDashboardPayload {
  const resolvedCustomerId = customerId.trim() || seedDeterministicUuid('customer', 3);
  const campaignOneId = seedDeterministicUuid('campaign', 11);
  const campaignTwoId = seedDeterministicUuid('campaign', 12);
  const campaignThreeId = seedDeterministicUuid('campaign', 13);

  const spendMicro = 38_472_650_000;
  const revenueMicro = 51_938_420_000;
  const profitMicro = revenueMicro - spendMicro;

  return {
    customer_id: resolvedCustomerId,
    period: { from, to },
    active: 7,
    paused: 2,
    archived: 1,
    impressions_7d: 184_732,
    clicks_7d: 29_417,
    unique_clicks_7d: 24_863,
    overspend_count: 1,
    kpis: {
      spend_micro: spendMicro,
      revenue_micro: revenueMicro,
      profit_micro: profitMicro,
      conversions: 1847,
      unique_clicks: 24_863,
      roi_pct: 34.87,
      freshness: {
        as_of: new Date().toISOString(),
        consistency: 'eventual',
        stale: false,
        freshness_label: 'Chart preview data (?chart_mock=1); not live API metrics',
        ch_lag_seconds: 0,
      },
    },
    series: buildSyntheticSeries(from, to),
    attention: [
      {
        id: campaignTwoId,
        name: 'Nordic Finance Lead Gen',
        reason: 'Pacing mode: asap',
      },
      {
        id: campaignThreeId,
        name: 'LATAM Install Burst',
        reason: 'Overspend risk',
      },
    ],
    breakdowns: {
      campaigns: {
        total: 3,
        truncated: false,
        rows: [
          {
            id: campaignOneId,
            name: 'US Search Retargeting',
            clicks: 11_284,
            conversions: 743,
            revenue_micro: 19_472_830_000,
            profit_micro: 4_827_190_000,
            roi_pct: 33.12,
          },
          {
            id: campaignTwoId,
            name: 'Nordic Finance Lead Gen',
            clicks: 9_137,
            conversions: 612,
            revenue_micro: 16_284_710_000,
            profit_micro: 3_918_440_000,
            roi_pct: 31.74,
          },
          {
            id: campaignThreeId,
            name: 'LATAM Install Burst',
            clicks: 8_996,
            conversions: 492,
            revenue_micro: 16_180_880_000,
            profit_micro: 3_274_620_000,
            roi_pct: 28.43,
          },
        ],
      },
    },
  };
}
