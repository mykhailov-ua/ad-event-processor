/** Wire ISO -> datetime-local input value (YYYY-MM-DDTHH:mm). */
export function toDatetimeLocalValue(iso: string | undefined): string {
  if (!iso) {
    return '';
  }
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) {
    return '';
  }
  return formatDatetimeLocalValue(parsed);
}

/** datetime-local value -> wire ISO. */
export function fromDatetimeLocalValue(value: string): string | undefined {
  if (!value.trim()) {
    return undefined;
  }
  const parsed = parseDatetimeLocalValue(value);
  if (!parsed) {
    return undefined;
  }
  return parsed.toISOString();
}

export function parseDatetimeLocalValue(value: string): Date | undefined {
  if (!value.trim()) {
    return undefined;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }
  return parsed;
}

export function formatDatetimeLocalValue(date: Date): string {
  const pad = (part: number) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function mergeDatetimeLocalDate(date: Date, hours: number, minutes: number): string {
  const merged = new Date(date);
  merged.setHours(hours, minutes, 0, 0);
  return formatDatetimeLocalValue(merged);
}

export function startOfDayLocalValue(date: Date): string {
  const normalized = new Date(date);
  normalized.setHours(0, 0, 0, 0);
  return formatDatetimeLocalValue(normalized);
}

export function endOfDayLocalValue(date: Date): string {
  const normalized = new Date(date);
  normalized.setHours(23, 59, 0, 0);
  return formatDatetimeLocalValue(normalized);
}

/** YYYY-MM-DD wire value -> local Date (date-only; no timezone shift). */
export function parseDateValue(value: string): Date | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(trimmed);
  if (!match) {
    return undefined;
  }
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  if (month < 1 || month > 12 || day < 1 || day > 31) {
    return undefined;
  }
  const parsed = new Date(year, month - 1, day);
  if (
    parsed.getFullYear() !== year ||
    parsed.getMonth() !== month - 1 ||
    parsed.getDate() !== day
  ) {
    return undefined;
  }
  return parsed;
}

/** Local Date -> YYYY-MM-DD wire value. */
export function formatDateValue(date: Date): string {
  const pad = (part: number) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

/** YYYY-MM wire value -> local Date (first day of month). */
export function parseMonthValue(value: string): Date | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const match = /^(\d{4})-(\d{2})$/.exec(trimmed);
  if (!match) {
    return undefined;
  }
  const year = Number(match[1]);
  const month = Number(match[2]);
  if (month < 1 || month > 12) {
    return undefined;
  }
  return new Date(year, month - 1, 1);
}

/** Local Date -> YYYY-MM wire value. */
export function formatMonthValue(date: Date): string {
  const pad = (part: number) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}`;
}
