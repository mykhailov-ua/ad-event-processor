import { formatDistanceToNow, isValid, parseISO } from 'date-fns';

const MICRO_PER_USD = 1_000_000;

// Display helpers for Cold directory cells (ui.mdc): prefer server *_display; format wire micro as USD.
/** Prefer server *_display fields; otherwise format wire micro integer as USD. */
export function displayMicro(value?: number | null, display?: string | null): string {
  const formatted = display?.trim();
  if (formatted) {
    return formatted;
  }
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  const usd = value / MICRO_PER_USD;
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(usd);
}

/** Prefer server *_display fields; fall back to wire count integer. */
export function displayCount(value?: number | null, display?: string | null): string {
  const formatted = display?.trim();
  if (formatted) {
    return formatted;
  }
  if (value == null) {
    return '';
  }
  return new Intl.NumberFormat('en-US').format(value);
}

/** Prefer server *_display fields; fall back to wire ISO timestamp. */
export function displayTimestamp(iso?: string | null, display?: string | null): string {
  const formatted = display?.trim();
  if (formatted) {
    return formatted.replace(/\s+/g, ' ');
  }
  const raw = iso?.trim();
  if (!raw) {
    return '';
  }
  return raw.replace(/\s+/g, ' ');
}

/** Relative time for list tooltips; falls back to absolute display when ISO is missing. */
export function displayRelativeTimestamp(iso?: string | null, display?: string | null): string {
  const raw = iso?.trim();
  if (raw) {
    const date = parseISO(raw);
    if (isValid(date)) {
      return formatDistanceToNow(date, { addSuffix: true });
    }
  }
  return displayTimestamp(iso, display);
}

/** Prefer server *_display fields; fall back to decimal USD wire string. */
export function displayMoneyDecimal(value?: string | null, display?: string | null): string {
  const formatted = display?.trim();
  if (formatted) {
    return formatted;
  }
  return value?.trim() ?? '';
}
