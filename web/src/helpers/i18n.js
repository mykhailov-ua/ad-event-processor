/** @type {Record<string, string>} */
const EN = {
  'nav.overview': 'Overview',
  'nav.campaigns': 'Campaigns',
  'nav.portfolio': 'Portfolio',
  'nav.billing': 'Billing',
  'nav.reports': 'Reports',
  'nav.ops': 'Operations',
  'nav.settings': 'Settings',
  'action.load': 'Load',
  'action.export': 'Export CSV',
  'status.loading': 'Loading…',
  'report.compare': 'Compare with previous period',
};

/**
 * Resolve a UI string from the English catalog.
 *
 * @param {string} key
 * @param {string} [fallback]
 * @returns {string}
 */
export function t(key, fallback = key) {
  return EN[key] ?? fallback;
}
