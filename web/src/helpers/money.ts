export function formatUsdDecimal(
  decimal: string | number | null | undefined,
  options: { currency?: boolean } = {}
): string {
  if (decimal == null || decimal === '') return '-';
  const num = Number(decimal);
  if (!Number.isFinite(num)) return String(decimal);
  const prefix = options.currency !== false ? '$' : '';
  return `${prefix}${num.toFixed(2)}`;
}

export function formatAmountMicro(micro: number | null | undefined, currency = 'USD'): string {
  if (micro == null || !Number.isFinite(micro)) return '-';
  const sign = micro < 0 ? '-' : '';
  const abs = Math.abs(micro);
  const dollars = Math.floor(abs / 1_000_000);
  const frac = abs % 1_000_000;
  const cents = Math.round(frac / 10_000);
  const amount = `${sign}${dollars}.${String(cents).padStart(2, '0')}`;
  return currency ? `${amount} ${currency}` : amount;
}
