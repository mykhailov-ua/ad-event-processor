export const DEFAULT_LIST_LIMIT = 50;

export const OPTIMAL_LIST_LIMIT_MAX = 100;

export function parseListLimit(raw: string | null, max = 500): number {
  if (!raw) {
    return DEFAULT_LIST_LIMIT;
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return DEFAULT_LIST_LIMIT;
  }
  return Math.min(parsed, max);
}

export function clampListLimit(value: number, max = OPTIMAL_LIST_LIMIT_MAX): number {
  if (!Number.isFinite(value) || value < 1) {
    return DEFAULT_LIST_LIMIT;
  }
  return Math.min(Math.max(1, Math.trunc(value)), max);
}

export function parseListOffset(raw: string | null): number {
  if (!raw) {
    return 0;
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }
  return parsed;
}
