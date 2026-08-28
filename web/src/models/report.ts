export type ReportKey = string;

export const RETIRED_REPORT_ALTS: Record<string, string> = {
  'campaign-unit-economics': '/reports/placements',
  'source-margin': '/reports/source-quality',
};

export const REPORT_ROUTE_PATHS: Record<ReportKey, string> = {
  placements: '/reports/placements',
  keywords: '/reports/keywords',
  'pacing-drift': '/reports/pacing-drift',
  'filter-rejects': '/reports/filter-rejects',
  'fraud-breakdown': '/reports/fraud-breakdown',
  'silent-reject-impression-funnel': '/reports/silent-reject-impression-funnel',
  'spend-velocity': '/reports/spend-velocity',
  'daypart-heatmap': '/reports/daypart-heatmap',
  'campaign-geo-device': '/reports/campaign-geo-device',
  'geo-roi': '/reports/geo-roi',
  'source-quality': '/reports/source-quality',
  'ivt-by-source': '/reports/ivt-by-source',
  'click-log': '/reports/clicks',
  'conversion-type-payout': '/reports/conversion-type-payout',
  'postback-reconciliation': '/reports/postback-reconciliation',
  'rtb-overview': '/reports/rtb/overview',
  'rtb-no-bid-reasons': '/reports/rtb/no-bid-reasons',
  'rtb-geo-device': '/reports/rtb/geo-device',
  'traffic-sources': '/reports/traffic-sources',
  'discrepancy-buy-sell': '/reports/discrepancy-buy-sell',
  'true-roi': '/reports/true-roi',
  'cost-sync-coverage': '/reports/cost-sync-coverage',
  'campaign-overview': '/reports/campaign-overview',
  'customer-portfolio': '/reports/customer-portfolio',
  'data-quality': '/reports/data-quality',
  'layer-desync-summary': '/reports/layer-desync-summary',
  'layer-desync-drilldown': '/reports/layer-desync-drilldown',
  'fraud-evidence-pack': '/reports/fraud-evidence-pack',
  'signal-effectiveness': '/reports/signal-effectiveness',
  'rtt-split-tunnel': '/reports/rtt-split-tunnel',
  'campaign-toggle-cohort': '/reports/campaign-toggle-cohort',
  'wire-signal-breakdown': '/reports/wire-signal-breakdown',
  'customer-fraud-by-type': '/reports/customer-fraud-by-type',
  'customer-fraud-by-dimension': '/reports/customer-fraud-by-dimension',
  'customer-fraud-evidence': '/reports/customer-fraud-evidence',
  'edge-parity': '/reports/edge-parity',
  'ml/feature-spikes': '/reports/ml/feature-spikes',
  'ml/score-distribution': '/reports/ml/score-distribution',
  'ml/shadow-delta': '/reports/ml/shadow-delta',
  telegram: '/reports/telegram',
  'telegram-funnel': '/reports/telegram/funnel',
  'telegram-bots': '/reports/telegram/bots',
  'telegram-premium': '/reports/telegram/premium',
  'telegram-fraud': '/reports/telegram/fraud',
};

const REPORT_TITLES: Record<ReportKey, string> = {
  placements: 'Placements',
  keywords: 'Keywords',
  'pacing-drift': 'Pacing drift',
  'filter-rejects': 'Filter rejects',
  'fraud-breakdown': 'Fraud breakdown',
  'silent-reject-impression-funnel': 'Silent reject impression funnel',
  'spend-velocity': 'Spend velocity',
  'daypart-heatmap': 'Daypart heatmap',
  'campaign-geo-device': 'Campaign geo device',
  'geo-roi': 'Geo ROI',
  'source-quality': 'Source quality',
  'ivt-by-source': 'IVT by source',
  'click-log': 'Click log',
  'conversion-type-payout': 'Conversion type payout',
  'postback-reconciliation': 'Postback reconciliation',
  'rtb-overview': 'RTB overview',
  'rtb-no-bid-reasons': 'RTB no-bid reasons',
  'rtb-geo-device': 'RTB geo device',
  'traffic-sources': 'Traffic sources',
  'discrepancy-buy-sell': 'Discrepancy buy sell',
  'true-roi': 'True ROI',
  'cost-sync-coverage': 'Cost sync coverage',
  'campaign-overview': 'Campaign overview',
  'customer-portfolio': 'Customer portfolio',
  'data-quality': 'Data quality',
  'layer-desync-summary': 'Layer desync summary',
  'layer-desync-drilldown': 'Layer desync drilldown',
  'fraud-evidence-pack': 'Fraud evidence pack',
  'signal-effectiveness': 'Signal effectiveness',
  'rtt-split-tunnel': 'RTT split tunnel',
  'campaign-toggle-cohort': 'Campaign toggle cohort',
  'wire-signal-breakdown': 'Wire signal breakdown',
  'customer-fraud-by-type': 'Customer fraud by type',
  'customer-fraud-by-dimension': 'Customer fraud by dimension',
  'customer-fraud-evidence': 'Customer fraud evidence',
  'edge-parity': 'Edge parity',
  'ml/feature-spikes': 'ML feature spikes',
  'ml/score-distribution': 'ML score distribution',
  'ml/shadow-delta': 'ML shadow delta',
  telegram: 'Telegram summary',
  'telegram-funnel': 'Telegram funnel',
  'telegram-bots': 'Telegram bots',
  'telegram-premium': 'Telegram premium',
  'telegram-fraud': 'Telegram fraud',
};

export const REPORT_CATALOG: Array<{ key: ReportKey; live?: boolean; title?: string }> = [
  { key: 'placements', live: true, title: 'Placements' },
  { key: 'keywords', live: true, title: 'Keywords' },
  { key: 'pacing-drift', live: true, title: 'Pacing drift' },
  { key: 'filter-rejects', live: true, title: 'Filter rejects' },
  { key: 'fraud-breakdown', live: true, title: 'Fraud breakdown' },
  { key: 'silent-reject-impression-funnel', live: true, title: 'Silent reject impression funnel' },
  { key: 'spend-velocity', live: true, title: 'Spend velocity' },
  { key: 'daypart-heatmap', live: true, title: 'Daypart heatmap' },
  { key: 'campaign-geo-device', live: true, title: 'Campaign geo device' },
  { key: 'geo-roi', live: true, title: 'Geo ROI' },
  { key: 'source-quality', live: true, title: 'Source quality' },
  { key: 'ivt-by-source', live: true, title: 'IVT by source' },
  { key: 'click-log', live: true, title: 'Click log' },
  { key: 'conversion-type-payout', live: true, title: 'Conversion type payout' },
  { key: 'postback-reconciliation', live: true, title: 'Postback reconciliation' },
  { key: 'rtb-overview', live: true, title: 'RTB overview' },
  { key: 'rtb-no-bid-reasons', live: true, title: 'RTB no-bid reasons' },
  { key: 'rtb-geo-device', live: true, title: 'RTB geo device' },
  { key: 'traffic-sources', live: true, title: 'Traffic sources' },
  { key: 'discrepancy-buy-sell', live: true, title: 'Discrepancy buy sell' },
  { key: 'true-roi', live: true, title: 'True ROI' },
  { key: 'cost-sync-coverage', live: true, title: 'Cost sync coverage' },
  { key: 'campaign-overview', live: true, title: 'Campaign overview' },
  { key: 'customer-portfolio', live: true, title: 'Customer portfolio' },
  { key: 'data-quality', live: true, title: 'Data quality' },
  { key: 'layer-desync-summary', live: true, title: 'Layer desync summary' },
  { key: 'layer-desync-drilldown', live: true, title: 'Layer desync drilldown' },
  { key: 'fraud-evidence-pack', live: true, title: 'Fraud evidence pack' },
  { key: 'signal-effectiveness', live: true, title: 'Signal effectiveness' },
  { key: 'rtt-split-tunnel', live: true, title: 'RTT split tunnel' },
  { key: 'campaign-toggle-cohort', live: true, title: 'Campaign toggle cohort' },
  { key: 'wire-signal-breakdown', live: true, title: 'Wire signal breakdown' },
  { key: 'customer-fraud-by-type', live: true, title: 'Customer fraud by type' },
  { key: 'customer-fraud-by-dimension', live: true, title: 'Customer fraud by dimension' },
  { key: 'customer-fraud-evidence', live: true, title: 'Customer fraud evidence' },
  { key: 'edge-parity', live: true, title: 'Edge parity' },
  { key: 'ml/feature-spikes', live: true, title: 'ML feature spikes' },
  { key: 'ml/score-distribution', live: true, title: 'ML score distribution' },
  { key: 'ml/shadow-delta', live: true, title: 'ML shadow delta' },
  { key: 'telegram', live: true, title: 'Telegram summary' },
  { key: 'telegram-funnel', live: true, title: 'Telegram funnel' },
  { key: 'telegram-bots', live: true, title: 'Telegram bots' },
  { key: 'telegram-premium', live: true, title: 'Telegram premium' },
  { key: 'telegram-fraud', live: true, title: 'Telegram fraud' },
];

export function isReportLive(key: string): boolean {
  return REPORT_CATALOG.some((entry) => entry.key === key && entry.live === true);
}

export function reportHref(key: string): string {
  const retired = RETIRED_REPORT_ALTS[key];
  if (retired) return retired;
  const route = REPORT_ROUTE_PATHS[key];
  if (route) return route;
  return `/reports/${key}`;
}

export function reportTitle(key: string, fallback?: string): string {
  if (REPORT_TITLES[key]) return REPORT_TITLES[key];
  if (fallback) return fallback;
  return key.replace(/-/g, ' ');
}

export function formatReportColumnLabel(columnKey: string): string {
  const normalized = columnKey.replace(/^ghost_/i, 'silent_reject_').replace(/_/g, ' ');
  return normalized.replace(/\b\w/g, (char) => char.toUpperCase());
}
