import { buildDashboardMockPortfolio } from '@/domains/dashboards/dashboard_series_mock';
import type { BuyerPortfolio } from '@/domains/dashboards/buyer_dashboard_types';

import { devMockStore } from './store.ts';
import { devMockIso, usdToMicro } from './fixture_helpers.ts';
import { seedDeterministicUuid } from './seed_uuid.ts';

function reportFreshness() {
  return { stale: false, label: 'Synthetic preview' };
}

export function devMockRoleDashboard(url: URL, pathname: string) {
  const role = pathname.slice('/api/v1/dashboards/'.length).split('/')[0];
  const customerId = url.searchParams.get('customer_id')?.trim();
  const from = url.searchParams.get('from') ?? devMockIso(7);
  const to = url.searchParams.get('to') ?? devMockIso(0);

  if (role === 'buyer') {
    const portfolio: BuyerPortfolio = buildDashboardMockPortfolio({
      customer_id: customerId,
      period: { from, to, timezone: 'UTC' },
      series: [],
      breakdowns: {
        campaigns: { rows: [] },
        landers: { rows: [] },
        offers: { rows: [] },
        sources: { rows: [] },
      },
      recent_clicks: [],
    });
    return portfolio;
  }

  if (role === 'campaign') {
    const campaignId = pathname.split('/').pop() ?? seedDeterministicUuid('campaign', 1);
    const campaign = devMockStore().campaigns.find((row) => row.id === campaignId);
    return buildDashboardMockPortfolio({
      customer_id: campaign?.customer_id ?? customerId,
      period: { from, to },
      series: [],
      campaigns: campaign
        ? [{ id: campaignId, name: campaign.name, status: campaign.status }]
        : undefined,
    });
  }

  const series = buildDashboardMockPortfolio({
    period: { from, to },
    series: [],
  }).series;

  const totals = {
    spend_micro: usdToMicro(84_200),
    revenue_micro: usdToMicro(112_450),
    profit_micro: usdToMicro(28_250),
    conversions: 1842,
    clicks: 286_400,
    impressions: 1_420_000,
    freshness: reportFreshness(),
  };

  if (role === 'cfo' || role === 'accountant') {
    return {
      role,
      period: { from, to },
      totals,
      series,
      tables: {
        invoices: {
          rows: [
            {
              month: '2026-08',
              invoiced_micro: usdToMicro(42_100),
              paid_micro: usdToMicro(39_800),
            },
            {
              month: '2026-07',
              invoiced_micro: usdToMicro(38_400),
              paid_micro: usdToMicro(38_400),
            },
          ],
        },
      },
      freshness: reportFreshness(),
    };
  }

  if (role === 'fraud') {
    return {
      role,
      period: { from, to },
      totals: {
        blocks: 12_840,
        silent_rejects: 3_420,
        ivt_rate_pct: 4.2,
        freshness: reportFreshness(),
      },
      series,
      breakdowns: {
        categories: {
          rows: [
            { id: 'bot', name: 'Automated bot', blocks: 6200 },
            { id: 'proxy', name: 'Proxy / VPN', blocks: 4100 },
            { id: 'datacenter', name: 'Datacenter ASN', blocks: 2540 },
          ],
        },
      },
    };
  }

  return {
    role,
    period: { from, to },
    totals,
    series,
    services: [
      { id: 'tracker', name: 'Tracker', status: 'ok', detail: 'Within SLA' },
      { id: 'processor', name: 'Processor', status: 'ok', detail: 'Within SLA' },
    ],
    freshness: reportFreshness(),
  };
}
