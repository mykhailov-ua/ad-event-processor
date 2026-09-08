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

/** OpenAPI reportClicks alias; catalog and SPA use click-log. */
export const REPORT_CATALOG_KEY_ALIASES: Record<string, string> = {
  clicks: 'click-log',
};

export function reportHubPath(key: string): string {
  if (key === 'click-log') {
    return '/reports/click-log';
  }
  if (key === 'rtb-overview' || key === 'rtb-no-bid-reasons' || key === 'rtb-geo-device') {
    return '/rtb';
  }
  if (key === 'telegram') {
    return '/reports/telegram';
  }
  if (key.startsWith('telegram/')) {
    return `/reports/telegram/${key.slice('telegram/'.length)}`;
  }
  if (key.includes('/')) {
    return `/reports/${encodeURIComponent(key)}`;
  }
  return `/reports/${encodeURIComponent(key)}`;
}

export const EVIDENCE_PACK_REPORT_KEYS = new Set([
  'customer-fraud-evidence',
  'fraud-evidence-pack',
]);

export const EXPORT_ONLY_REPORT_KEYS = new Set(['fraud-evidence-pack-bulk']);

/** Catalog keys with dedicated typed admin pages (registered before reports/:key). */
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
]);

export const TYPED_RTB_REPORT_KEYS = new Set([
  'rtb-overview',
  'rtb-no-bid-reasons',
  'rtb-geo-device',
]);

export function typedReportRedirectPath(key: string): string | undefined {
  if (TYPED_RTB_REPORT_KEYS.has(key)) {
    return '/rtb';
  }
  if (TYPED_CUSTOMER_REPORT_KEYS.has(key)) {
    return reportHubPath(key);
  }
  return undefined;
}

export function isTypedCatalogReportKey(key: string): boolean {
  return TYPED_CUSTOMER_REPORT_KEYS.has(key) || TYPED_RTB_REPORT_KEYS.has(key);
}

export const CAMPAIGN_STATS_REPORT_KEYS = new Set(['campaign-stats']);

export function resolveReportCatalogKey(key: string): string {
  return REPORT_CATALOG_KEY_ALIASES[key] ?? key;
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
  const search = new URLSearchParams();
  if (params.reportKey?.trim()) {
    search.set('report_key', params.reportKey.trim());
  }
  if (params.customerId?.trim()) {
    search.set('customer_id', params.customerId.trim());
  }
  if (params.from?.trim()) {
    search.set('from', params.from.trim());
  }
  if (params.to?.trim()) {
    search.set('to', params.to.trim());
  }
  if (params.format?.trim()) {
    search.set('format', params.format.trim());
  }
  if (params.jobId?.trim()) {
    search.set('job_id', params.jobId.trim());
  }
  const query = search.toString();
  return query ? `/reports/jobs?${query}` : '/reports/jobs';
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
