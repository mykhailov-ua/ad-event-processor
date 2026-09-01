import { useCallback, useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import { CalendarIcon } from 'lucide-react';

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
};

type RangeEdge = 'from' | 'to';

function formatRangeLabel(from: Date | undefined, to: Date | undefined): string {
  if (!from) {
    return 'Pick date range';
  }
  if (!to) {
    return format(from, 'MMM d, yyyy');
  }
  if (from.getFullYear() === to.getFullYear()) {
    return `${format(from, 'MMM d')} – ${format(to, 'MMM d, yyyy')}`;
  }
  return `${format(from, 'MMM d, yyyy')} – ${format(to, 'MMM d, yyyy')}`;
}

function formatEdgeLabel(value: Date | undefined): string {
  if (!value) {
    return '—';
  }
  return format(value, 'MMM d, yyyy');
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

export function DateRangePicker({
  id,
  label,
  from,
  to,
  onChange,
  disabled = false,
  className,
}: DateRangePickerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [monthCount, setMonthCount] = useState(resolveMonthCount);
  const [activeEdge, setActiveEdge] = useState<RangeEdge>('from');
  const [draftFrom, setDraftFrom] = useState<Date | undefined>();
  const [draftTo, setDraftTo] = useState<Date | undefined>();
  const [visibleMonth, setVisibleMonth] = useState<Date>(() => new Date());

  const fromDate = useMemo(() => parseDatetimeLocalValue(from), [from]);
  const toDate = useMemo(() => parseDatetimeLocalValue(to), [to]);
  const displayLabel = formatRangeLabel(fromDate, toDate);

  const selectedRange = useMemo(() => {
    if (!draftFrom) {
      return undefined;
    }
    return { from: draftFrom, to: draftTo };
  }, [draftFrom, draftTo]);

  function resetDraft() {
    setDraftFrom(fromDate ? normalizePickerDay(fromDate) : undefined);
    setDraftTo(toDate ? normalizePickerDay(toDate) : undefined);
    setActiveEdge('from');
    setVisibleMonth(fromDate ?? toDate ?? new Date());
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      setAlign(resolvePopoverAlign(triggerRef.current));
      setMonthCount(resolveMonthCount());
      resetDraft();
    }
    setOpen(nextOpen);
  }

  function commitRange(nextFrom: Date | undefined, nextTo: Date | undefined) {
    if (!nextFrom || !nextTo) {
      return;
    }
    const orderedFrom = nextFrom <= nextTo ? nextFrom : nextTo;
    const orderedTo = nextFrom <= nextTo ? nextTo : nextFrom;
    onChange(startOfDayLocalValue(orderedFrom), endOfDayLocalValue(orderedTo));
    setOpen(false);
  }

  const handleDayClick = useCallback(
    (day: Date) => {
      const picked = normalizePickerDay(day);

      if (activeEdge === 'from') {
        setDraftFrom(picked);
        setDraftTo((currentTo) => (currentTo && picked > currentTo ? undefined : currentTo));
        setActiveEdge('to');
        return;
      }

      setDraftFrom((currentFrom) => {
        if (!currentFrom) {
          return picked;
        }
        if (picked < currentFrom) {
          setDraftTo(currentFrom);
          return picked;
        }
        setDraftTo(picked);
        return currentFrom;
      });
    },
    [activeEdge],
  );

  function handleApply() {
    commitRange(
      draftFrom ? normalizePickerDay(draftFrom) : undefined,
      draftTo ? normalizePickerDay(draftTo) : undefined,
    );
  }

  return (
    <div className={cn('grid gap-2', className)}>
      <Label htmlFor={id}>{label}</Label>
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
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
        </PopoverTrigger>
        <PopoverContent
          align={align}
          className="ui-date-range-picker p-0 [&_.ui-shell]:!w-auto [&_.ui-shell]:!min-w-0"
          side="bottom"
        >
          <div className="flex gap-2 border-b border-border/50 p-3">
            <Button
              type="button"
              shape="square"
              variant={activeEdge === 'from' ? 'secondary' : 'outline'}
              size="sm"
              className="h-auto min-h-10 flex-1 flex-col items-start gap-0.5 rounded-xl px-3 py-2"
              onClick={() => setActiveEdge('from')}
            >
              <span className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                Start
              </span>
              <span className="truncate text-sm font-normal tabular-nums">{formatEdgeLabel(draftFrom)}</span>
            </Button>
            <Button
              type="button"
              shape="square"
              variant={activeEdge === 'to' ? 'secondary' : 'outline'}
              size="sm"
              className="h-auto min-h-10 flex-1 flex-col items-start gap-0.5 rounded-xl px-3 py-2"
              disabled={!draftFrom}
              onClick={() => setActiveEdge('to')}
            >
              <span className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                End
              </span>
              <span className="truncate text-sm font-normal tabular-nums">{formatEdgeLabel(draftTo)}</span>
            </Button>
          </div>
          <div className="overflow-x-auto px-3 pb-3 pt-2">
            <Calendar
              mode="range"
              numberOfMonths={monthCount}
              selected={selectedRange}
              onSelect={(range, triggerDate) => {
                if (range?.from && range.to) {
                  setDraftFrom(normalizePickerDay(range.from));
                  setDraftTo(normalizePickerDay(range.to));
                  setActiveEdge('to');
                  return;
                }
                handleDayClick(triggerDate);
              }}
              month={visibleMonth}
              onMonthChange={setVisibleMonth}
              className="mx-auto w-max rounded-xl bg-muted/25 p-2 [--cell-size:2.25rem] [--picker-radius-control:calc(var(--radius)-0.25rem)]"
              classNames={{
                months: 'relative flex flex-col gap-6 md:flex-row md:gap-8',
                month: 'relative w-[calc(var(--cell-size)*7+1.5rem)] space-y-3',
                month_caption: 'relative z-[1] flex h-[--cell-size] items-center justify-center',
                nav: 'absolute inset-x-0 top-0 z-[2] flex items-center justify-between',
                weekdays: 'flex w-full gap-1',
                weekday: 'flex-1 text-center text-[0.75rem] font-normal text-muted-foreground',
                week: 'mt-1 flex w-full gap-1',
                day: 'relative flex flex-1 items-center justify-center p-0.5',
              }}
            />
          </div>
          <div className="flex justify-end gap-2 border-t border-border/50 p-3">
            <Button
              type="button"
              shape="square"
              variant="ghost"
              size="sm"
              className="rounded-xl"
              onClick={() => {
                onChange('', '');
                setOpen(false);
              }}
            >
              Clear
            </Button>
            <Button
              type="button"
              shape="square"
              size="sm"
              className="rounded-xl"
              disabled={!draftFrom || !draftTo}
              onClick={handleApply}
            >
              Apply
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
