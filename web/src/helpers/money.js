/**
 * @typedef {number} MoneyMicro - integer micro-units (1 USD = 1_000_000 micro)
 */

const MICRO = 1_000_000;
const MAX_FRAC_DIGITS = 6;

/**
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
 * @param {MoneyMicro} micro
 * @returns {string} formatted to 2 decimal places
 */
export function formatMicro(micro) {
  const abs = Math.abs(micro);
  const dollars = Math.floor(abs / MICRO);
  const cents = Math.round((abs % MICRO) / 10_000);
  const sign = micro < 0 ? '-' : '';
  return `${sign}${dollars}.${String(cents).padStart(2, '0')}`;
}

/**
 * @param {MoneyMicro} micro
 * @returns {string} formatted to 6 decimal places
 */
export function formatMicroFull(micro) {
  const abs = Math.abs(micro);
  const dollars = Math.floor(abs / MICRO);
  const frac = abs % MICRO;
  const sign = micro < 0 ? '-' : '';
  return `${sign}${dollars}.${String(frac).padStart(6, '0')}`;
}

/**
 * @param {number} micro
 * @param {string} [currency]
 * @returns {string}
 */
export function formatAmountMicro(micro, currency = '') {
  const amt = formatMicro(micro);
  return currency ? `${amt} ${currency}` : amt;
}

/**
 * @param {string} decimal
 * @returns {string}
 */
export function formatDecimalDisplay(decimal) {
  if (!decimal) return '—';
  return decimal;
}

/**
 * @param {string | number | null | undefined} decimal
 * @returns {string}
 */
export function formatUsdDecimal(decimal) {
  if (decimal == null || decimal === '') return '—';
  return formatDecimalDisplay(String(decimal));
}
