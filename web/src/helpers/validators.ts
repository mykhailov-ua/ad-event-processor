const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const MAX_RANGE_MS = 90 * 24 * 60 * 60 * 1000;

/**
 * Validate a report date range and return an error message or null.
 */
export function validateReportRange(from: string, to: string): string | null {
  const f = new Date(from);
  const t = new Date(to);
  if (Number.isNaN(f.getTime())) return 'invalid from date';
  if (Number.isNaN(t.getTime())) return 'invalid to date';
  if (f >= t) return 'from must be before to';
  if (t.getTime() - f.getTime() > MAX_RANGE_MS) return 'range exceeds 90 days';
  return null;
}

/**
 * Test whether a string is a valid UUID.
 */
export function validateUuid(s: string): boolean {
  return typeof s === 'string' && UUID_RE.test(s);
}

/**
 * Validate a customer id field value.
 */
export function validateCustomerIdField(value: string): string | null {
  const trimmed = String(value).trim();
  if (!trimmed) return 'Customer UUID is required';
  if (!validateUuid(trimmed)) return 'Invalid UUID format';
  return null;
}

/**
 * Validate a three-letter ISO currency code.
 */
export function validateCurrency(code: string): string | null {
  const trimmed = String(code).trim();
  if (!/^[A-Z]{3}$/.test(trimmed)) return 'Currency must be a 3-letter ISO code (e.g. USD)';
  return null;
}

/**
 * Validate a tracking domain field value.
 */
export function validateTrackingDomain(domain: string): string | null {
  if (!String(domain).trim()) return 'Tracking domain is required';
  return null;
}

/**
 * Validate a self-serve budget in micro-units against min and max bounds.
 */
export function validateSelfServeBudget(micro: number, minMicro: number, maxMicro: number): string | null {
  if (!Number.isInteger(micro) || micro <= 0) return 'budget must be positive';
  if (micro < minMicro) return `budget must be at least ${minMicro} micro`;
  if (micro > maxMicro) return `budget exceeds maximum ${maxMicro} micro`;
  return null;
}
