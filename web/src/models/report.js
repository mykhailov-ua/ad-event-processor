/**
 * @typedef {{ key: string, title: string, live?: boolean, buyer?: boolean }} ReportCardDTO
 */

/** @type {ReportCardDTO[]} */
export const REPORT_CATALOG = [
  { key: 'placements', title: 'Placements', live: true, buyer: true },
  { key: 'keywords', title: 'Keywords', live: true, buyer: true },
  { key: 'pacing-drift', title: 'Pacing drift', buyer: true },
  { key: 'spend-velocity', title: 'Spend velocity' },
  { key: 'daypart-heatmap', title: 'Daypart heatmap', buyer: true },
  { key: 'campaign-geo-device', title: 'Geo & device', buyer: true },
  { key: 'geo-roi', title: 'Geo ROI', live: true, buyer: true },
  { key: 'source-quality', title: 'Source quality', buyer: true },
  { key: 'ivt-by-source', title: 'IVT by source', live: true },
  { key: 'postback-reconciliation', title: 'Postback reconciliation' },
  { key: 'traffic-sources', title: 'Traffic sources', live: true, buyer: true },
  { key: 'discrepancy-buy-sell', title: 'Buy/sell discrepancy' },
  { key: 'campaign-unit-economics', title: 'Unit economics' },
  { key: 'source-margin', title: 'Source margin' },
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
  'postback-reconciliation': '/api/v1/reports/postback-reconciliation',
  'pacing-drift': '/api/v1/reports/pacing-drift',
  'spend-velocity': '/api/v1/reports/spend-velocity',
  'campaign-unit-economics': '/api/v1/reports/campaign-unit-economics',
  'source-margin': '/api/v1/reports/source-margin',
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
 * Build client route href for a report key.
 *
 * @param {string} key
 * @returns {string}
 */
export function reportHref(key) {
  return `/reports/${key}`;
}
