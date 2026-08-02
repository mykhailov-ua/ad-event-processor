const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const MAX_RANGE_MS = 90 * 24 * 60 * 60 * 1000;

/**
 * @param {string} from ISO8601
 * @param {string} to ISO8601
 * @returns {string|null} error message or null
 */
export function validateReportRange(from, to) {
  const f = new Date(from);
  const t = new Date(to);
  if (isNaN(f.getTime())) return 'invalid from date';
  if (isNaN(t.getTime())) return 'invalid to date';
  if (f >= t) return 'from must be before to';
  if (t - f > MAX_RANGE_MS) return 'range exceeds 90 days';
  return null;
}

/**
 * @param {string} s
 * @returns {boolean}
 */
export function validateUuid(s) {
  return typeof s === 'string' && UUID_RE.test(s);
}

/**
 * @param {string} value
 */
export function validateCustomerIdField(value) {
  const trimmed = String(value).trim();
  if (!trimmed) return 'Customer UUID is required';
  if (!validateUuid(trimmed)) return 'Invalid UUID format';
  return null;
}

/**
 * @param {string} code
 */
export function validateCurrency(code) {
  const trimmed = String(code).trim();
  if (!/^[A-Z]{3}$/.test(trimmed)) return 'Currency must be a 3-letter ISO code (e.g. USD)';
  return null;
}

/**
 * @param {string} domain
 */
export function validateTrackingDomain(domain) {
  if (!String(domain).trim()) return 'Tracking domain is required';
  return null;
}

/**
 * @param {number} micro
 * @param {number} minMicro
 * @param {number} maxMicro
 * @returns {string|null} error message or null
 */
export function validateSelfServeBudget(micro, minMicro, maxMicro) {
  if (!Number.isInteger(micro) || micro <= 0) return 'budget must be positive';
  if (micro < minMicro) return `budget must be at least ${minMicro} micro`;
  if (micro > maxMicro) return `budget exceeds maximum ${maxMicro} micro`;
  return null;
}
