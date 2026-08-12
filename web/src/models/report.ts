export type ReportCardDTO = {
  key: string;
  title: string;
  /** Semantic icon id for nav + reports hub (see web/scripts/icon_phosphor_map.json). */
  icon: string;
  live?: boolean;
  retired?: boolean;
  buyer?: boolean;
};

export const RETIRED_REPORT_ALTS: Record<string, { href: string; label: string }> = {
  'pacing-drift': { href: '/campaigns/portfolio', label: 'Portfolio (drift sort)' },
  'campaign-unit-economics': { href: '/reports/placements', label: 'Placements' },
  'source-margin': { href: '/reports/placements', label: 'Placements' },
  'postback-reconciliation': { href: '/billing', label: 'Billing ledger' },
};

export const REPORT_CATALOG: ReportCardDTO[] = [
  { key: 'placements', title: 'Placements', icon: 'grid-four', live: true, buyer: true },
  { key: 'keywords', title: 'Keywords', icon: 'tag', live: true, buyer: true },
  { key: 'pacing-drift', title: 'Pacing drift', icon: 'chart-line-down', retired: true, buyer: true },
  { key: 'spend-velocity', title: 'Spend velocity', icon: 'speedometer', live: true },
  { key: 'daypart-heatmap', title: 'Daypart heatmap', icon: 'clock', buyer: true, live: true },
  { key: 'campaign-geo-device', title: 'Geo & device', icon: 'devices', buyer: true, live: true },
  { key: 'geo-roi', title: 'Geo ROI', icon: 'map-pin', live: true, buyer: true },
  { key: 'source-quality', title: 'Source quality', icon: 'star', buyer: true, live: true },
  { key: 'ivt-by-source', title: 'IVT by source', icon: 'shield-alert', live: true },
  { key: 'postback-reconciliation', title: 'Postback reconciliation', icon: 'arrows-left-right', retired: true },
  { key: 'traffic-sources', title: 'Traffic sources', icon: 'funnel', live: true, buyer: true },
  { key: 'discrepancy-buy-sell', title: 'Buy/sell discrepancy', icon: 'scales', live: true },
  { key: 'true-roi', title: 'True ROI', icon: 'currency-dollar', live: true, buyer: true },
  { key: 'campaign-unit-economics', title: 'Unit economics', icon: 'calculator', retired: true },
  { key: 'source-margin', title: 'Source margin', icon: 'percent', retired: true },
  { key: 'customer-portfolio', title: 'Customer portfolio', icon: 'briefcase', live: true },
  { key: 'campaign-overview', title: 'Campaign overview', icon: 'presentation-chart', live: true },
  { key: 'telegram', title: 'Telegram Mini Apps', icon: 'send', live: true, buyer: true },
];

export const STUB_REPORT_PATHS: Record<string, string> = {
  'telegram': '/api/v1/reports/telegram',
};

/**
 * Resolve display title for a report route key.
 */
export function reportTitle(key: string): string {
  for (let i = 0; i < REPORT_CATALOG.length; i++) {
    if (REPORT_CATALOG[i].key === key) return REPORT_CATALOG[i].title;
  }
  return key || 'Report';
}

/**
 * Resolve sidebar / hub icon for a report route key.
 */
export function reportIcon(key: string): string {
  for (let i = 0; i < REPORT_CATALOG.length; i++) {
    if (REPORT_CATALOG[i].key === key) return REPORT_CATALOG[i].icon;
  }
  return 'file-text';
}

/**
 * Resolve stub report API path from route key.
 */
export function stubReportPath(key: string): string | null {
  return STUB_REPORT_PATHS[key] ?? null;
}

/**
 * Return true when a report key was retired in favor of a live alternative.
 */
export function isRetiredReport(key: string): boolean {
  return key in RETIRED_REPORT_ALTS;
}

/**
 * Resolve retired report redirect target.
 */
export function retiredReportAlt(key: string): { href: string; label: string } | null {
  return RETIRED_REPORT_ALTS[key] ?? null;
}

/**
 * Build client route href for a report key.
 */
export function reportHref(key: string): string {
  return `/reports/${key}`;
}
