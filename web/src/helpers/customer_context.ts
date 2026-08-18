import * as storage from './storage.js';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function isCustomerUuid(value: string): boolean {
  return UUID_RE.test(String(value).trim());
}

export function shortCustomerId(id: string, len = 8): string {
  const s = String(id);
  return s.length > len + 1 ? `${s.slice(0, len)}…` : s;
}

export function touchCustomerContext(id: string): void {
  const trimmed = String(id).trim();
  if (!isCustomerUuid(trimmed)) return;
  storage.pushRecentCustomerId(trimmed);
}
