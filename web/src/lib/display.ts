import { formatDistanceToNow, isValid, parseISO } from 'date-fns';

/** Prefer server *_display fields; fall back to wire micro integer. */
export function displayMicro(value?: number | null, display?: string | null): string {
  const formatted = display?.trim();
  if (formatted) {
    return formatted;
  }
  if (value == null) {
    return '';
  }
  return String(value);
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
