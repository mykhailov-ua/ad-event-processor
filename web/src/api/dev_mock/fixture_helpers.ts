export function devMockIso(daysAgo = 0, hoursAgo = 0): string {
  const ms = Date.now() - daysAgo * 86_400_000 - hoursAgo * 3_600_000;
  return new Date(ms).toISOString();
}

export function devMockMonth(offsetMonths = 0): string {
  const date = new Date();
  date.setUTCDate(1);
  date.setUTCMonth(date.getUTCMonth() + offsetMonths);
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

export function usdToMicro(usd: number): number {
  return Math.round(usd * 1_000_000);
}

export function microDisplay(micro: number, currency = 'USD'): string {
  const amount = micro / 1_000_000;
  return `${currency} ${amount.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

export function countDisplay(value: number): string {
  return value.toLocaleString('en-US');
}

export function slicePage<T>(
  items: T[],
  limit: number,
  offset: number
): { items: T[]; total: number } {
  const safeLimit = Math.max(1, limit);
  const safeOffset = Math.max(0, offset);
  return {
    items: items.slice(safeOffset, safeOffset + safeLimit),
    total: items.length,
  };
}

export function parseLimitOffset(url: URL, defaultLimit = 50): { limit: number; offset: number } {
  const limit =
    Number.parseInt(url.searchParams.get('limit') ?? String(defaultLimit), 10) || defaultLimit;
  const offset = Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0;
  return { limit, offset };
}
