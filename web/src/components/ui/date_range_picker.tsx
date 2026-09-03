import { useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import { CalendarIcon } from 'lucide-react';
import type { DateRange } from 'react-day-picker';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  endOfDayLocalValue,
  parseDatetimeLocalValue,
  startOfDayLocalValue,
} from '@/lib/datetime_range';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { cn } from '@/lib/utils';

export type DateRangePickerProps = {
  id: string;
  label: string;
  from: string;
  to: string;
  onChange: (from: string, to: string) => void;
  disabled?: boolean;
  className?: string;
  variant?: 'default' | 'admin';
};

function formatRangeLabel(from: Date | undefined, to: Date | undefined): string {
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

function resolveMonthCount(): number {
  if (typeof window === 'undefined') {
    return 2;
  }
  return window.matchMedia('(min-width: 768px)').matches ? 2 : 1;
}

function normalizePickerDay(day: Date): Date {
  const normalized = new Date(day);
  normalized.setHours(0, 0, 0, 0);
  return normalized;
}

function toDraftRange(from: Date | undefined, to: Date | undefined): DateRange | undefined {
  if (!from) {
    return undefined;
  }
  return { from: normalizePickerDay(from), to: to ? normalizePickerDay(to) : undefined };
}

export function DateRangePicker({
  id,
  label,
  from,
  to,
  onChange,
  disabled = false,
  className,
  variant = 'default',
}: DateRangePickerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [monthCount, setMonthCount] = useState(resolveMonthCount);
  const [draftRange, setDraftRange] = useState<DateRange | undefined>();

  const fromDate = useMemo(() => parseDatetimeLocalValue(from), [from]);
  const toDate = useMemo(() => parseDatetimeLocalValue(to), [to]);
  const displayLabel = formatRangeLabel(fromDate, toDate);

  function resetDraft() {
    setDraftRange(toDraftRange(fromDate, toDate));
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      setAlign(resolvePopoverAlign(triggerRef.current));
      setMonthCount(resolveMonthCount());
      resetDraft();
    }
    setOpen(nextOpen);
  }

  function handleApply() {
    if (!draftRange?.from || !draftRange.to) {
      return;
    }
    const orderedFrom =
      draftRange.from <= draftRange.to ? draftRange.from : draftRange.to;
    const orderedTo = draftRange.from <= draftRange.to ? draftRange.to : draftRange.from;
    onChange(startOfDayLocalValue(orderedFrom), endOfDayLocalValue(orderedTo));
    setOpen(false);
  }

  const trigger = (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        {variant === 'admin' ? (
          <button
            ref={triggerRef}
            id={id}
            type="button"
            disabled={disabled}
            className={cn(
              'relative w-full flex h-8 w-full items-center justify-between rounded-md border border-zinc-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950 inline-flex max-w-full items-center gap-2',
              !fromDate && 'text-zinc-500 dark:text-zinc-400',
            )}
          >
            <CalendarIcon className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
            <span className="truncate">{displayLabel}</span>
          </button>
        ) : (
          <Button
            ref={triggerRef}
            id={id}
            type="button"
            variant="outline"
            disabled={disabled}
            className={cn(
              'h-9 w-full justify-start text-left font-normal',
              !fromDate && 'text-muted-foreground',
            )}
          >
            <CalendarIcon className="mr-2 h-4 w-4 shrink-0" />
            <span className="truncate">{displayLabel}</span>
          </Button>
        )}
      </PopoverTrigger>
      <PopoverContent
        align={align}
        className={cn(
          'w-auto p-0 [&_.ui-shell]:!w-auto [&_.ui-shell]:!min-w-0 [&_.ui-shell-panel]:overflow-visible',
          variant === 'admin' && 'rounded-md border border-zinc-200 bg-white p-0 shadow-lg dark:border-zinc-800 dark:bg-zinc-950',
        )}
        side="bottom"
      >
        <div className={variant === 'admin' ? 'p-3' : 'p-3'}>
          <Calendar
            mode="range"
            numberOfMonths={monthCount}
            selected={draftRange}
            defaultMonth={draftRange?.from ?? fromDate ?? new Date()}
            variant={variant === 'admin' ? 'admin' : 'default'}
            onSelect={setDraftRange}
          />
        </div>
        <div
          className={
            variant === 'admin'
              ? 'flex justify-end gap-2 border-t border-zinc-200 p-2 dark:border-zinc-800'
              : 'flex justify-end gap-2 border-t border-border/50 px-3 py-3'
          }
        >
          <button
            className="inline-flex h-8 items-center justify-center gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm font-medium transition active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 h-auto min-h-8 leading-normal"
            type="button"
            onClick={() => {
              onChange('', '');
              setOpen(false);
            }}
          >
            Clear
          </button>
          <button
            className="inline-flex h-8 items-center justify-center gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm font-medium transition active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 border border-zinc-900 bg-zinc-900 text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200 h-auto min-h-8 leading-normal"
            disabled={!draftRange?.from || !draftRange?.to}
            type="button"
            onClick={handleApply}
          >
            Apply
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );

  if (variant === 'admin') {
    return (
      <label className={cn('text-sm font-medium text-zinc-700 dark:text-zinc-300', className)}>
        {label}
        {trigger}
      </label>
    );
  }

  return (
    <div className={cn('grid gap-2', className)}>
      <Label htmlFor={id}>{label}</Label>
      {trigger}
    </div>
  );
}
