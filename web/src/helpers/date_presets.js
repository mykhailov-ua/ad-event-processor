/**
 * Return the current instant as an ISO-8601 string.
 *
 * @returns {string}
 */
export function toIsoNow() {
  return new Date().toISOString();
}

/**
 * Return an ISO-8601 timestamp for midnight N days ago.
 *
 * @param {number} days
 * @returns {string}
 */
export function isoDaysAgo(days) {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString();
}

/**
 * Return an ISO-8601 timestamp for the start of the current month.
 *
 * @returns {string}
 */
export function isoMonthStart() {
  const d = new Date();
  d.setDate(1);
  d.setHours(0, 0, 0, 0);
  return d.toISOString();
}

/** Report range quick presets. */
export const REPORT_DATE_PRESETS = [
  { id: '7d', label: '7 days', from: () => isoDaysAgo(7), to: () => toIsoNow() },
  { id: '30d', label: '30 days', from: () => isoDaysAgo(30), to: () => toIsoNow() },
  { id: 'mtd', label: 'MTD', from: () => isoMonthStart(), to: () => toIsoNow() },
];
