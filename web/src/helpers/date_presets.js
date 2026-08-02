/**
 * @returns {string}
 */
export function toIsoNow() {
  return new Date().toISOString();
}

/**
 * @param {number} days
 */
export function isoDaysAgo(days) {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString();
}

/**
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
