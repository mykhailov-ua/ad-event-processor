const EN: Record<string, string> = {
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
 */
export function t(key: string, fallback = key): string {
  return EN[key] ?? fallback;
}
