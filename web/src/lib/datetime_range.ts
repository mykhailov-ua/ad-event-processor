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
