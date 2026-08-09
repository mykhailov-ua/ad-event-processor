/**
 * @typedef {number} MoneyMicro - integer micro-units (1 USD = 1_000_000 micro)
 */

const MICRO = 1_000_000;
const MAX_FRAC_DIGITS = 6;

/**
 * Parse a decimal string into integer micro-units.
 *
 * @param {string} s
 * @returns {MoneyMicro}
 * @throws {Error} if invalid
 */
export function ParseDecimal(s) {
  if (typeof s !== 'string' || s.trim() === '') throw new Error('invalid decimal: empty');
  const trimmed = s.trim();
  if (!/^-?\d+(\.\d+)?$/.test(trimmed)) throw new Error(`invalid decimal: ${s}`);

  const [intPart, fracPart = ''] = trimmed.split('.');
  if (fracPart.length > MAX_FRAC_DIGITS) throw new Error(`too many fractional digits: ${s}`);

  const intVal = BigInt(intPart) * BigInt(MICRO);
  let fracVal = BigInt(0);
  if (fracPart) {
    const padded = fracPart.padEnd(MAX_FRAC_DIGITS, '0');
    fracVal = BigInt(padded);
  }

  const micro = intPart.startsWith('-') ? intVal - fracVal : intVal + fracVal;
  if (micro > BigInt(Number.MAX_SAFE_INTEGER) || micro < BigInt(-Number.MAX_SAFE_INTEGER)) {
    throw new Error(`overflow: ${s}`);
  }
  return Number(micro);
}

/**
 * Format micro-units as a dollar amount with two decimal places by default ($00.00).
 *
 * @param {MoneyMicro} micro
 * @returns {string}
 */
export function formatMicro(micro) {
  const abs = Math.abs(micro);
  const dollars = Math.floor(abs / MICRO);
  const cents = Math.round((abs % MICRO) / 10_000);
  const sign = micro < 0 ? '-' : '';
  return `${sign}${dollars}.${String(cents).padStart(2, '0')}`;
}

/**
 * Format micro-units as a dollar amount with six decimal places ($00.000000).
 *
 * @param {MoneyMicro} micro
 * @returns {string}
 */
export function formatMicroFull(micro) {
  const abs = Math.abs(micro);
  const dollars = Math.floor(abs / MICRO);
  const frac = abs % MICRO;
  const sign = micro < 0 ? '-' : '';
  return `${sign}${dollars}.${String(frac).padStart(6, '0')}`;
}

/**
 * Format micro-units with an optional currency suffix.
 *
 * @param {number} micro
 * @param {string} [currency]
 * @returns {string}
 */
export function formatAmountMicro(micro, currency = '') {
  const amt = formatMicro(micro);
  return currency ? `${amt} ${currency}` : amt;
}

/**
 * Format a decimal string for display, using an em dash when empty.
 *
 * @param {string} decimal
 * @returns {string}
 */
export function formatDecimalDisplay(decimal) {
  if (!decimal) return '—';
  return decimal;
}

/**
 * Format a USD decimal field for display.
 * Default: "$00.00" (dollars & cents, e.g. "$100.00").
 * Options: { full: true } -> "$100.000000" for detailed view.
 *
 * @param {string | number | null | undefined} decimal
 * @param {{ full?: boolean, currency?: boolean }} [options]
 * @returns {string}
 */
export function formatUsdDecimal(decimal, options = {}) {
  if (decimal == null || decimal === '') return '—';
  const num = Number(decimal);
  if (!Number.isFinite(num)) return String(decimal);
  const prefix = options.currency !== false ? '$' : '';
  if (options.full) {
    return `${prefix}${num.toFixed(6)}`;
  }
  return `${prefix}${num.toFixed(2)}`;
}
