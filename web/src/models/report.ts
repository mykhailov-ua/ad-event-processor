export type ReportCardDTO = {
  key: string;
  title: string;

  icon: string;
  live?: boolean;
  retired?: boolean;
  buyer?: boolean;
};

export const RETIRED_REPORT_ALTS: Record<string, { href: string; label: string; title: string }> = {
  'campaign-unit-economics': {
    href: '/reports/placements',
    label: 'Placements',
    title: 'Unit economics retired; use Placements for spend, ROI, and IVT by zone.',
  },
  'source-margin': {
    href: '/reports/source-quality',
    label: 'Source quality',
    title: 'Source margin retired; use Source quality for sub-source IVT and quality scores.',
  },
};

export const REPORT_CATALOG: ReportCardDTO[] = [
  { key: 'placements', title: 'Placements', icon: 'grid-four', live: true, buyer: true },
  { key: 'keywords', title: 'Keywords', icon: 'tag', live: true, buyer: true },
  { key: 'pacing-drift', title: 'Pacing drift', icon: 'chart-line-down', live: true, buyer: true },
  { key: 'filter-rejects', title: 'Filter rejects', icon: 'prohibit', live: true },
  { key: 'fraud-breakdown', title: 'Fraud breakdown', icon: 'shield-warning', live: true },
  {
    key: 'ghost-impression-funnel',
    title: 'Ghost impression funnel',
    icon: 'eye',
    live: true,
  },
  { key: 'spend-velocity', title: 'Spend velocity', icon: 'speedometer', live: true },
  { key: 'daypart-heatmap', title: 'Daypart heatmap', icon: 'clock', buyer: true, live: true },
  { key: 'campaign-geo-device', title: 'Geo & device', icon: 'devices', buyer: true, live: true },
  { key: 'geo-roi', title: 'Geo ROI', icon: 'map-pin', live: true, buyer: true },
  { key: 'source-quality', title: 'Source quality', icon: 'star', buyer: true, live: true },
  { key: 'ivt-by-source', title: 'IVT by source', icon: 'shield-alert', live: true },
  {
    key: 'postback-reconciliation',
    title: 'Postback reconciliation',
    icon: 'arrows-left-right',
    live: true,
  },
  { key: 'rtb-overview', title: 'RTB overview', icon: 'broadcast', live: true },
  { key: 'rtb-no-bid-reasons', title: 'RTB no-bid reasons', icon: 'x-circle', live: true },
  { key: 'rtb-geo-device', title: 'RTB geo & device', icon: 'globe', live: true },
  { key: 'traffic-sources', title: 'Traffic sources', icon: 'funnel', live: true, buyer: true },
  { key: 'discrepancy-buy-sell', title: 'Buy/sell discrepancy', icon: 'scales', live: true },
  { key: 'true-roi', title: 'True ROI', icon: 'currency-dollar', live: true, buyer: true },
  { key: 'cost-sync-coverage', title: 'Cost sync coverage', icon: 'cloud-arrow-down', live: true, buyer: true },
  { key: 'campaign-unit-economics', title: 'Unit economics', icon: 'calculator', retired: true },
  { key: 'source-margin', title: 'Source margin', icon: 'percent', retired: true },
  { key: 'customer-portfolio', title: 'Customer portfolio', icon: 'briefcase', live: true },
  { key: 'data-quality', title: 'Data quality', icon: 'database', live: true },
  { key: 'campaign-overview', title: 'Campaign overview', icon: 'presentation-chart', live: true },
  { key: 'telegram', title: 'Telegram Mini Apps', icon: 'send', live: true, buyer: true },
];

export const STUB_REPORT_PATHS: Record<string, string> = {
  telegram: '/api/v1/reports/telegram',
};

export const REPORT_PATH_OVERRIDES: Record<string, string> = {
  'rtb-overview': '/reports/rtb/overview',
  'rtb-no-bid-reasons': '/reports/rtb/no-bid-reasons',
  'rtb-geo-device': '/reports/rtb/geo-device',
};

export function reportTitle(key: string): string {
  for (let i = 0; i < REPORT_CATALOG.length; i++) {
    if (REPORT_CATALOG[i].key === key) return REPORT_CATALOG[i].title;
  }
  return key || 'Report';
}

export function reportIcon(key: string): string {
  for (let i = 0; i < REPORT_CATALOG.length; i++) {
    if (REPORT_CATALOG[i].key === key) return REPORT_CATALOG[i].icon;
  }
  return 'file-text';
}

export function stubReportPath(key: string): string | null {
  return STUB_REPORT_PATHS[key] ?? null;
}

export function isRetiredReport(key: string): boolean {
  return key in RETIRED_REPORT_ALTS;
}

export function retiredReportAlt(key: string): { href: string; label: string; title: string } | null {
  return RETIRED_REPORT_ALTS[key] ?? null;
}

export function reportHref(key: string): string {
  return REPORT_PATH_OVERRIDES[key] ?? `/reports/${key}`;
}
