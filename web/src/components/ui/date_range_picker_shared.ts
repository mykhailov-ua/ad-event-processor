import { format } from 'date-fns';
import type { DateRange } from 'react-day-picker';

export function formatFooterRange(from: Date | undefined, to: Date | undefined): string {
  if (!from) {
    return '';
  }
  if (!to) {
    return format(from, 'MMM d, yyyy');
  }
  return `${format(from, 'MMM d, yyyy')} - ${format(to, 'MMM d, yyyy')}`;
}

export function formatRangeLabel(from: Date | undefined, to: Date | undefined): string {
  if (!from) {
    return 'Pick date range';
  }
  if (!to) {
    return format(from, 'MMM d, yyyy');
  }
  if (from.getFullYear() === to.getFullYear()) {
    return `${format(from, 'MMM d')} - ${format(to, 'MMM d, yyyy')}`;
  }
  return `${format(from, 'MMM d, yyyy')} - ${format(to, 'MMM d, yyyy')}`;
}

export function resolveMonthCount(): number {
  if (typeof window === 'undefined') {
    return 2;
  }
  return window.matchMedia('(min-width: 768px)').matches ? 2 : 1;
}

export function estimateToolbarDateRangePopoverWidth(monthCount: number): number {
  return monthCount * 272 + 48;
}

export function normalizePickerDay(day: Date): Date {
  const normalized = new Date(day);
  normalized.setHours(0, 0, 0, 0);
  return normalized;
}

export function toDraftRange(from: Date | undefined, to: Date | undefined): DateRange | undefined {
  if (!from) {
    return undefined;
  }
  return { from: normalizePickerDay(from), to: to ? normalizePickerDay(to) : undefined };
}
