import { buildExportHubHref } from '@/lib/export_hub_paths';

const REPORT_KEY_PATH_OVERRIDES: Record<string, string> = {
  clicks: '/api/v1/reports/clicks',
  'rtb-overview': '/api/v1/reports/rtb/overview',
  'rtb-no-bid-reasons': '/api/v1/reports/rtb/no-bid-reasons',
  'rtb-geo-device': '/api/v1/reports/rtb/geo-device',
  'ml/feature-spikes': '/api/v1/reports/ml/feature-spikes',
  'ml/score-distribution': '/api/v1/reports/ml/score-distribution',
  'ml/shadow-delta': '/api/v1/reports/ml/shadow-delta',
  telegram: '/api/v1/reports/telegram',
  'telegram/summary': '/api/v1/reports/telegram/summary',
  'telegram/funnel': '/api/v1/reports/telegram/funnel',
  'telegram/bots': '/api/v1/reports/telegram/bots',
  'telegram/premium': '/api/v1/reports/telegram/premium',
  'telegram/fraud': '/api/v1/reports/telegram/fraud',
};

export const REPORT_CATALOG_KEY_ALIASES: Record<string, string> = {
  clicks: 'click-log',
  'ghost-impression-funnel': 'silent-reject-impression-funnel',
};

export const TYPED_EXPORT_ONLY_REPORT_KEYS = new Set(['fraud-evidence-pack-bulk']);

export const EXPORT_ONLY_REPORT_KEYS = TYPED_EXPORT_ONLY_REPORT_KEYS;

export const TYPED_CUSTOMER_REPORT_KEYS = new Set([
  'click-log',
  'customer-fraud-by-type',
  'postback-reconciliation',
  'source-quality',
  'placements',
  'keywords',
  'geo-roi',
  'traffic-sources',
  'data-quality',
  'pacing-drift',
  'fraud-breakdown',
  'wire-signal-breakdown',
  'silent-reject-impression-funnel',
  'signal-effectiveness',
  'customer-fraud-by-dimension',
  'ivt-by-source',
  'layer-desync-summary',
  'layer-desync-drilldown',
  'rtt-split-tunnel',
  'campaign-toggle-cohort',
  'filter-rejects',
  'conversion-type-payout',
  'campaign-overview',
  'campaign-geo-device',
  'spend-velocity',
  'daypart-heatmap',
  'true-roi',
  'cost-sync-coverage',
  'customer-portfolio',
  'discrepancy-buy-sell',
]);

export const TYPED_TELEGRAM_REPORT_KEYS = new Set([
  'telegram',
  'telegram/summary',
  'telegram/funnel',
  'telegram/bots',
  'telegram/premium',
  'telegram/fraud',
]);

export const TYPED_OPS_REPORT_KEYS = new Set(['edge-parity']);

export const TYPED_ML_REPORT_KEYS = new Set([
  'ml/feature-spikes',
  'ml/score-distribution',
  'ml/shadow-delta',
]);

export const TYPED_CAMPAIGN_STATS_REPORT_KEYS = new Set(['campaign-stats']);

export const TYPED_EVIDENCE_PACK_REPORT_KEYS = new Set([
  'customer-fraud-evidence',
  'fraud-evidence-pack',
]);

export const TYPED_RTB_REPORT_KEYS = new Set([
  'rtb-overview',
  'rtb-no-bid-reasons',
  'rtb-geo-device',
]);

export function isTypedCatalogReportKey(key: string): boolean {
  const resolved = resolveReportCatalogKey(key);
  return (
    TYPED_CUSTOMER_REPORT_KEYS.has(resolved) ||
    TYPED_RTB_REPORT_KEYS.has(resolved) ||
    TYPED_TELEGRAM_REPORT_KEYS.has(resolved) ||
    TYPED_OPS_REPORT_KEYS.has(resolved) ||
    TYPED_ML_REPORT_KEYS.has(resolved) ||
    TYPED_CAMPAIGN_STATS_REPORT_KEYS.has(resolved) ||
    TYPED_EVIDENCE_PACK_REPORT_KEYS.has(resolved) ||
    TYPED_EXPORT_ONLY_REPORT_KEYS.has(resolved)
  );
}

export function resolveReportCatalogKey(key: string): string {
  return REPORT_CATALOG_KEY_ALIASES[key] ?? key;
}

const REPORT_STUB_PATH_ALIASES: Record<string, string> = {
  'click-log': 'clicks',
  'silent-reject-impression-funnel': 'ghost-impression-funnel',
};

export function normalizeReportRouteSplat(splat: string | undefined): string | undefined {
  const trimmed = splat?.trim();
  if (!trimmed) {
    return undefined;
  }
  const decoded = decodeURIComponent(trimmed).replace(/^\/+|\/+$/g, '');
  return decoded || undefined;
}

export function reportStubPathFromKey(reportKey: string): string {
  const resolved = resolveReportCatalogKey(reportKey);
  return REPORT_STUB_PATH_ALIASES[resolved] ?? resolved;
}

export function reportRequiresCustomerScope(reportKey: string): boolean {
  const resolved = resolveReportCatalogKey(reportKey);
  if (!isTypedCatalogReportKey(resolved)) {
    return false;
  }
  if (TYPED_RTB_REPORT_KEYS.has(resolved) || TYPED_OPS_REPORT_KEYS.has(resolved)) {
    return false;
  }
  return true;
}

export function resolveReportStubRequiresCustomer(catalogKey: string): boolean {
  const resolved = resolveReportCatalogKey(catalogKey);
  if (isTypedCatalogReportKey(resolved)) {
    return reportRequiresCustomerScope(resolved);
  }
  return false;
}

const REPORT_TITLE_ACRONYMS = new Set(['roi', 'ivt', 'ml', 'rtb', 'rtt', 'kpi', 'cpa', 'csv']);

const REPORT_TITLE_OVERRIDES: Record<string, string> = {
  'campaign-geo-device': 'Campaign geo and device',
  'campaign-overview': 'Campaign overview',
  'campaign-stats': 'Campaign stats',
  'campaign-toggle-cohort': 'Campaign toggle cohort',
  'click-log': 'Click log',
  'conversion-type-payout': 'Conversion type payout',
  'cost-sync-coverage': 'Cost sync coverage',
  'customer-fraud-by-dimension': 'Fraud by dimension',
  'customer-fraud-by-type': 'Fraud by type',
  'customer-fraud-evidence': 'Dispute evidence',
  'customer-portfolio': 'Customer portfolio',
  'data-quality': 'Data quality',
  'daypart-heatmap': 'Daypart heatmap',
  'discrepancy-buy-sell': 'Buy vs sell discrepancy',
  'edge-parity': 'Edge parity',
  'filter-rejects': 'Filter rejects',
  'fraud-breakdown': 'Fraud breakdown',
  'fraud-evidence-pack': 'Fraud evidence pack',
  'fraud-evidence-pack-bulk': 'Fraud evidence pack bulk',
  'geo-roi': 'Geo ROI',
  'ivt-by-source': 'IVT by source',
  keywords: 'Keywords',
  'layer-desync-drilldown': 'Layer desync drilldown',
  'layer-desync-summary': 'Layer desync summary',
  'ml/feature-spikes': 'ML feature spikes',
  'ml/score-distribution': 'ML score distribution',
  'ml/shadow-delta': 'ML shadow delta',
  'pacing-drift': 'Pacing drift',
  placements: 'Placements',
  'postback-reconciliation': 'Postback reconciliation',
  'rtb-geo-device': 'RTB geo and device',
  'rtb-no-bid-reasons': 'RTB no-bid reasons',
  'rtb-overview': 'RTB overview',
  'rtt-split-tunnel': 'RTT split tunnel',
  'signal-effectiveness': 'Signal effectiveness',
  'silent-reject-impression-funnel': 'Non-blocking fraud response funnel',
  'source-quality': 'Source quality',
  'spend-velocity': 'Spend velocity',
  telegram: 'Telegram Mini App rollup',
  'telegram/bots': 'Telegram bot performance',
  'telegram/fraud': 'Telegram fraud signals',
  'telegram/funnel': 'Telegram conversion funnel',
  'telegram/premium': 'Telegram premium users',
  'telegram/summary': 'Telegram summary KPIs',
  'traffic-sources': 'Traffic sources',
  'true-roi': 'True ROI',
  'wire-signal-breakdown': 'Wire signal breakdown',
};

function humanizeReportKeyPart(part: string): string {
  const lower = part.toLowerCase();
  if (REPORT_TITLE_ACRONYMS.has(lower)) {
    return lower.toUpperCase();
  }
  return lower.charAt(0).toUpperCase() + lower.slice(1);
}

export function reportTitleFromKey(reportKey: string): string {
  const normalized = resolveReportCatalogKey(reportKey);
  const override = REPORT_TITLE_OVERRIDES[normalized];
  if (override) {
    return override;
  }
  return normalized.split(/[/-]/).filter(Boolean).map(humanizeReportKeyPart).join(' ');
}

export function resolveReportDisplayTitle(reportKey: string, catalogTitle?: string | null): string {
  const fromCatalog = catalogTitle?.trim();
  if (fromCatalog && fromCatalog !== reportKey) {
    return fromCatalog;
  }
  return reportTitleFromKey(reportKey);
}

export type ReportJobsHrefParams = {
  reportKey?: string;
  customerId?: string;
  from?: string;
  to?: string;
  format?: string;
  jobId?: string;
};

export function buildReportJobsHref(params: ReportJobsHrefParams = {}): string {
  return buildExportHubHref({
    reportKey: params.reportKey,
    customerId: params.customerId,
    from: params.from,
    to: params.to,
    format: params.format,
    jobId: params.jobId,
    kind: 'report',
  });
}

export function reportKeyToApiPath(key: string): string {
  const override = REPORT_KEY_PATH_OVERRIDES[key];
  if (override) {
    return override;
  }
  return `/api/v1/reports/${encodeURIComponent(key)}`;
}

export function defaultReportRange(defaultRange?: string): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to);
  const normalized = (defaultRange ?? '7d').trim().toLowerCase();
  const match = /^(\d+)([dh])$/.exec(normalized);

  if (match) {
    const amount = Number.parseInt(match[1], 10);
    if (match[2] === 'd') {
      from.setUTCDate(from.getUTCDate() - amount);
    } else {
      from.setUTCHours(from.getUTCHours() - amount);
    }
  } else {
    from.setUTCDate(from.getUTCDate() - 7);
  }

  return { from: from.toISOString(), to: to.toISOString() };
}

export function allTypedCatalogReportKeys(): Set<string> {
  const keys = new Set<string>();
  for (const set of [
    TYPED_CUSTOMER_REPORT_KEYS,
    TYPED_RTB_REPORT_KEYS,
    TYPED_TELEGRAM_REPORT_KEYS,
    TYPED_OPS_REPORT_KEYS,
    TYPED_ML_REPORT_KEYS,
    TYPED_CAMPAIGN_STATS_REPORT_KEYS,
    TYPED_EVIDENCE_PACK_REPORT_KEYS,
    TYPED_EXPORT_ONLY_REPORT_KEYS,
  ]) {
    for (const key of set) {
      keys.add(key);
    }
  }
  return keys;
}
