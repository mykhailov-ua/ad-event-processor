import { DEV_MOCK_CUSTOMERS } from './fixtures.ts';
import { devMockStore } from './store.ts';
import { devMockIso, usdToMicro } from './fixture_helpers.ts';
import {
  seedClickId,
  seedPlacementId,
  seedCatalogName,
  SEED_TRAFFIC_SOURCES,
} from './fixture_names.ts';

const REPORT_CATALOG_ROWS = [
  {
    key: 'campaign-overview',
    title: 'Campaign overview',
    description: 'Spend, conversions, and margin by campaign.',
    category: 'campaigns',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'click-log',
    title: 'Click log',
    description: 'Searchable click and postback events.',
    category: 'traffic',
    default_range: '7d',
    export_formats: ['csv', 'ndjson'],
    license_gated: false,
  },
  {
    key: 'clicks',
    title: 'Clicks',
    description: 'Aggregated click metrics.',
    category: 'traffic',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'geo-roi',
    title: 'GEO ROI',
    description: 'Country-level ROI and spend.',
    category: 'campaigns',
    default_range: '30d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'fraud-breakdown',
    title: 'Fraud breakdown',
    description: 'Blocks and silent rejects by category.',
    category: 'fraud',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'silent-reject-impression-funnel',
    title: 'Silent reject funnel',
    description: 'Impression to silent reject funnel.',
    category: 'fraud',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'rtb/overview',
    title: 'RTB overview',
    description: 'Auction volume and win rate.',
    category: 'rtb',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'rtb/no-bid-reasons',
    title: 'RTB no-bid reasons',
    description: 'Top no-bid reason codes.',
    category: 'rtb',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'traffic-sources',
    title: 'Traffic sources',
    description: 'Source quality and conversion rate.',
    category: 'traffic',
    default_range: '30d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'pacing-drift',
    title: 'Pacing drift',
    description: 'Budget pacing vs actual spend.',
    category: 'campaigns',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'postback-reconciliation',
    title: 'Postback reconciliation',
    description: 'Inbound vs outbound postback parity.',
    category: 'integrations',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'cost-sync-coverage',
    title: 'Cost sync coverage',
    description: 'Network cost sync completeness.',
    category: 'integrations',
    default_range: '30d',
    export_formats: ['csv'],
    license_gated: false,
  },
  {
    key: 'telegram/summary',
    title: 'Telegram summary',
    description: 'Mini-app funnel summary.',
    category: 'telegram',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: true,
    feature_key: 'telegram',
  },
] as const;

export function devMockReportCatalog() {
  return { rows: [...REPORT_CATALOG_ROWS] };
}

const COUNTRIES = ['US', 'DE', 'GB', 'BR', 'CA', 'PL', 'AU', 'IN'];

function buildReportRows(reportKey: string) {
  const campaigns = devMockStore().campaigns.slice(0, 12);
  return campaigns.map((campaign, index) => {
    const clicks = 1200 + index * 340;
    const conversions = Math.max(8, Math.round(clicks * (0.028 + index * 0.0015)));
    const spendMicro = usdToMicro(420 + index * 88);
    const revenueMicro = usdToMicro(610 + index * 124);
    const base = {
      campaign_id: campaign.id,
      campaign_name: campaign.name,
      country: COUNTRIES[index % COUNTRIES.length],
      clicks,
      conversions,
      spend_micro: spendMicro,
      revenue_micro: revenueMicro,
      profit_micro: revenueMicro - spendMicro,
      roi_pct: spendMicro > 0 ? ((revenueMicro - spendMicro) / spendMicro) * 100 : 0,
    };
    if (reportKey.includes('rtb')) {
      return {
        ...base,
        bids: clicks * 14,
        wins: Math.round(clicks * 0.62),
        no_bid_reason: index % 3 === 0 ? 'floor' : index % 3 === 1 ? 'geo' : 'category',
      };
    }
    if (reportKey.includes('fraud') || reportKey.includes('silent-reject')) {
      return {
        ...base,
        blocks: Math.round(clicks * 0.04),
        silent_rejects: Math.round(clicks * 0.012),
        fraud_category: index % 2 === 0 ? 'bot' : 'proxy',
      };
    }
    return base;
  });
}

export function devMockReportEnvelope(reportKey: string) {
  const rows = buildReportRows(reportKey);
  return {
    rows,
    total: rows.length,
    freshness: { stale: false, label: 'Synthetic preview' },
  };
}

export function devMockClickLogReport(url: URL) {
  const customerId = url.searchParams.get('customer_id') ?? DEV_MOCK_CUSTOMERS[0].id;
  const campaigns = devMockStore().campaigns.filter((row) => row.customer_id === customerId);
  const events = Array.from({ length: 25 }, (_, index) => {
    const campaign =
      campaigns[index % Math.max(campaigns.length, 1)] ?? devMockStore().campaigns[0];
    const hasRevenue = index % 3 !== 0;
    return {
      event_type: 'click',
      click_id: seedClickId(index + 1),
      campaign_id: campaign.id,
      placement_id: seedPlacementId(index + 1),
      created_at: devMockIso(0, index * 2),
      country: COUNTRIES[index % COUNTRIES.length],
      sub1: seedCatalogName(SEED_TRAFFIC_SOURCES, index + 1),
      goal_name: ['lead', 'sale', 'install'][index % 3],
      attributed_cost_micro: usdToMicro(0.12 + index * 0.03),
      revenue_micro: hasRevenue ? usdToMicro(18 + index * 4.5) : 0,
      inbound_status: 'accepted',
    };
  });
  const postbacks = events.slice(0, 8).map((event, index) => ({
    click_id: event.click_id,
    provider: ['webhook', 'facebook', 'google'][index % 3],
    status: index % 4 === 0 ? 'failed' : 'delivered',
    created_at: event.created_at,
    response_code: index % 4 === 0 ? 502 : 200,
  }));
  return {
    events,
    postbacks,
    freshness: { stale: false, label: 'Synthetic preview' },
    next_cursor: events.length >= 25 ? 'cursor-mock-2' : undefined,
  };
}
