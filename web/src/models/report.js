/**
 * @typedef {{ key: string, title: string, live?: boolean, retired?: boolean, buyer?: boolean }} ReportCardDTO
 */

/** @type {Record<string, { href: string, label: string }>} */
export const RETIRED_REPORT_ALTS = {
  'pacing-drift': { href: '/campaigns/portfolio', label: 'Portfolio (drift sort)' },
  'campaign-unit-economics': { href: '/reports/placements', label: 'Placements' },
  'source-margin': { href: '/reports/placements', label: 'Placements' },
  'postback-reconciliation': { href: '/billing', label: 'Billing ledger' },
};

/** @type {ReportCardDTO[]} */
export const REPORT_CATALOG = [
  { key: 'placements', title: 'Placements', live: true, buyer: true },
  { key: 'keywords', title: 'Keywords', live: true, buyer: true },
  { key: 'pacing-drift', title: 'Pacing drift', retired: true, buyer: true },
  { key: 'spend-velocity', title: 'Spend velocity' },
  { key: 'daypart-heatmap', title: 'Daypart heatmap', buyer: true },
  { key: 'campaign-geo-device', title: 'Geo & device', buyer: true },
  { key: 'geo-roi', title: 'Geo ROI', live: true, buyer: true },
  { key: 'source-quality', title: 'Source quality', buyer: true },
  { key: 'ivt-by-source', title: 'IVT by source', live: true },
  { key: 'postback-reconciliation', title: 'Postback reconciliation', retired: true },
  { key: 'traffic-sources', title: 'Traffic sources', live: true, buyer: true },
  { key: 'discrepancy-buy-sell', title: 'Buy/sell discrepancy' },
  { key: 'campaign-unit-economics', title: 'Unit economics', retired: true },
  { key: 'source-margin', title: 'Source margin', retired: true },
  { key: 'customer-portfolio', title: 'Customer portfolio' },
  { key: 'telegram', title: 'Telegram Mini Apps', live: true, buyer: true },
];

/** @type {Record<string, string>} */
export const STUB_REPORT_PATHS = {
  'daypart-heatmap': '/api/v1/reports/daypart-heatmap',
  'campaign-geo-device': '/api/v1/reports/campaign-geo-device',
  'geo-roi': '/api/v1/reports/geo-roi',
  'source-quality': '/api/v1/reports/source-quality',
  'ivt-by-source': '/api/v1/reports/ivt-by-source',
  'spend-velocity': '/api/v1/reports/spend-velocity',
  'traffic-sources': '/api/v1/reports/traffic-sources',
  'discrepancy-buy-sell': '/api/v1/reports/discrepancy-buy-sell',
  'campaign-overview': '/api/v1/reports/campaign-overview',
  'customer-portfolio': '/api/v1/reports/customer-portfolio',
  'telegram': '/api/v1/reports/telegram',
};

/**
 * Resolve display title for a report route key.
 *
 * @param {string} key
 * @returns {string}
 */
export function reportTitle(key) {
  for (let i = 0; i < REPORT_CATALOG.length; i++) {
    if (REPORT_CATALOG[i].key === key) return REPORT_CATALOG[i].title;
  }
  return key || 'Report';
}

/**
 * Resolve stub report API path from route key.
 *
 * @param {string} key
 * @returns {string|null}
 */
export function stubReportPath(key) {
  return STUB_REPORT_PATHS[key] ?? null;
}

/**
 * Return true when a report key was retired in favor of a live alternative.
 *
 * @param {string} key
 * @returns {boolean}
 */
export function isRetiredReport(key) {
  return key in RETIRED_REPORT_ALTS;
}

/**
 * Resolve retired report redirect target.
 *
 * @param {string} key
 * @returns {{ href: string, label: string }|null}
 */
export function retiredReportAlt(key) {
  return RETIRED_REPORT_ALTS[key] ?? null;
}

/**
 * Build client route href for a report key.
 *
 * @param {string} key
 * @returns {string}
 */
export function reportHref(key) {
  return `/reports/${key}`;
}
