/**
 * Human-readable labels for technical enum values in the admin UI.
 * API payloads keep machine values; only the display layer uses this module.
 */

/** @type {Record<string, string>} */
const EXACT = {
  // Deployment profiles
  single_vps: 'Single VPS',
  compose_dev: 'Docker Compose (development)',
  k8s_k3s: 'Kubernetes (K3s)',

  // Ingress / traffic
  espx_native: 'BidShard native',
  openrtb_3: 'OpenRTB 3.0',

  // Campaign status
  ACTIVE: 'Active',
  PAUSED: 'Paused',
  ARCHIVED: 'Archived',
  DRAFT: 'Draft',
  DELETED: 'Deleted',

  // Pacing
  EVEN: 'Even delivery',
  ASAP: 'As fast as possible',
  even: 'Even delivery',
  asap: 'As fast as possible',

  // Service health
  ok: 'OK',
  OK: 'OK',
  pass: 'Pass',
  PASS: 'Pass',
  fail: 'Fail',
  FAIL: 'Fail',
  unknown: 'Unknown',
  UNKNOWN: 'Unknown',
  disabled: 'Disabled',
  DISABLED: 'Disabled',
  degraded: 'Degraded',
  pending: 'Pending',
  processed: 'Processed',
  failed: 'Failed',
  live: 'Live',
  planned: 'Planned',
  active: 'Active',
  initialized: 'Initialized',

  // Invoice / ledger
  paid: 'Paid',
  draft: 'Draft',
  void: 'Void',
  voided: 'Voided',
  finalized: 'Finalized',
  open: 'Open',

  // Common services (doctor / ops)
  management: 'Management API',
  tracker: 'Tracker',
  postgres: 'PostgreSQL',
  clickhouse: 'ClickHouse',
  processor: 'Stream processor',
  control: 'Control plane',
  redis: 'Redis',
  nginx: 'Edge (Nginx)',
  kernel: 'Kernel',
  sysctl: 'Kernel limits',

  // Booleans
  yes: 'Yes',
  no: 'No',
  true: 'Yes',
  false: 'No',
};

const ACRONYMS = new Set(['vps', 'xdp', 'rtb', 'ivt', 'api', 'url', 'uuid', 'cpa', 'roi', 'rps', 'pg', 'ch']);

/**
 * Title-case one token, preserving known acronyms.
 *
 * @param {string} word
 * @returns {string}
 */
function titleWord(word) {
  const w = word.toLowerCase();
  if (w === 'k3s') return 'K3s';
  if (ACRONYMS.has(w)) return w.toUpperCase();
  return w.charAt(0).toUpperCase() + w.slice(1);
}

/**
 * Convert snake_case or SCREAMING_SNAKE_CASE to a readable phrase.
 *
 * @param {string} value
 * @returns {string}
 */
export function humanizeToken(value) {
  const s = String(value).trim();
  if (!s) return '—';
  return s
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .split(/[_\s]+/)
    .filter(Boolean)
    .map(titleWord)
    .join(' ');
}

/**
 * Resolve a human-readable label for a technical value.
 *
 * @param {string|number|boolean|null|undefined} value
 * @param {string} [fallback='—']
 * @returns {string}
 */
export function displayLabel(value, fallback = '—') {
  if (value == null || value === '') return fallback;
  const s = String(value);
  if (EXACT[s] != null) return EXACT[s];
  const lower = s.toLowerCase();
  if (EXACT[lower] != null) return EXACT[lower];
  return humanizeToken(s);
}

/**
 * @param {boolean|null|undefined} value
 * @returns {string}
 */
export function formatYesNo(value) {
  if (value == null) return '—';
  return value ? 'Yes' : 'No';
}

/**
 * Build select options with readable labels while preserving API values.
 *
 * @param {string[]} values
 * @returns {Array<{ value: string, label: string }>}
 */
export function labeledOptions(values) {
  return values.map((value) => ({ value, label: displayLabel(value) }));
}

export const PROFILE_SELECT_OPTIONS = labeledOptions(['single_vps', 'compose_dev', 'k8s_k3s']);
export const INGRESS_SELECT_OPTIONS = labeledOptions(['espx_native', 'openrtb_3']);

/** Major ISO currencies for settings (RUB excluded). */
export const CURRENCY_SELECT_OPTIONS = [
  { value: 'USD', label: 'US Dollar (USD)' },
  { value: 'EUR', label: 'Euro (EUR)' },
  { value: 'GBP', label: 'British Pound (GBP)' },
  { value: 'CHF', label: 'Swiss Franc (CHF)' },
  { value: 'JPY', label: 'Japanese Yen (JPY)' },
  { value: 'CAD', label: 'Canadian Dollar (CAD)' },
  { value: 'AUD', label: 'Australian Dollar (AUD)' },
  { value: 'CNY', label: 'Chinese Yuan (CNY)' },
  { value: 'UAH', label: 'Ukrainian Hryvnia (UAH)' },
  { value: 'PLN', label: 'Polish Złoty (PLN)' },
  { value: 'TRY', label: 'Turkish Lira (TRY)' },
  { value: 'KZT', label: 'Kazakhstani Tenge (KZT)' },
  { value: 'BRL', label: 'Brazilian Real (BRL)' },
  { value: 'MXN', label: 'Mexican Peso (MXN)' },
  { value: 'INR', label: 'Indian Rupee (INR)' },
  { value: 'SGD', label: 'Singapore Dollar (SGD)' },
  { value: 'HKD', label: 'Hong Kong Dollar (HKD)' },
  { value: 'AED', label: 'UAE Dirham (AED)' },
  { value: 'ILS', label: 'Israeli Shekel (ILS)' },
  { value: 'SEK', label: 'Swedish Krona (SEK)' },
  { value: 'NOK', label: 'Norwegian Krone (NOK)' },
  { value: 'DKK', label: 'Danish Krone (DKK)' },
];

/**
 * Currency select options, preserving a saved code outside the default list.
 *
 * @param {string} [current]
 * @returns {Array<{ value: string, label: string }>}
 */
export function currencySelectOptions(current = '') {
  const code = String(current || '').trim().toUpperCase();
  const known = new Set(CURRENCY_SELECT_OPTIONS.map((o) => o.value));
  if (code && !known.has(code)) {
    return [{ value: code, label: `${code} (saved)` }, ...CURRENCY_SELECT_OPTIONS];
  }
  return CURRENCY_SELECT_OPTIONS;
}

/** Common IANA timezones for platform settings. */
export const TIMEZONE_SELECT_OPTIONS = [
  { value: 'UTC', label: 'UTC — Coordinated Universal Time' },
  { value: 'Europe/London', label: 'Europe/London' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin' },
  { value: 'Europe/Paris', label: 'Europe/Paris' },
  { value: 'Europe/Kyiv', label: 'Europe/Kyiv' },
  { value: 'Europe/Warsaw', label: 'Europe/Warsaw' },
  { value: 'Europe/Istanbul', label: 'Europe/Istanbul' },
  { value: 'Asia/Dubai', label: 'Asia/Dubai' },
  { value: 'Asia/Kolkata', label: 'Asia/Kolkata' },
  { value: 'Asia/Singapore', label: 'Asia/Singapore' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo' },
  { value: 'Australia/Sydney', label: 'Australia/Sydney' },
  { value: 'America/New_York', label: 'America/New_York (Eastern)' },
  { value: 'America/Chicago', label: 'America/Chicago (Central)' },
  { value: 'America/Denver', label: 'America/Denver (Mountain)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (Pacific)' },
  { value: 'America/Sao_Paulo', label: 'America/Sao_Paulo' },
];

/**
 * Timezone select options, preserving a saved zone outside the default list.
 *
 * @param {string} [current]
 * @returns {Array<{ value: string, label: string }>}
 */
export function timezoneSelectOptions(current = '') {
  const tz = String(current || '').trim();
  const known = new Set(TIMEZONE_SELECT_OPTIONS.map((o) => o.value));
  if (tz && !known.has(tz)) {
    return [{ value: tz, label: `${tz} (saved)` }, ...TIMEZONE_SELECT_OPTIONS];
  }
  return TIMEZONE_SELECT_OPTIONS;
}
