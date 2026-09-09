import { useMemo, useRef, useState } from 'react';
import { CalendarIcon } from 'lucide-react';
import type { DateRange } from 'react-day-picker';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import {
  formatRangeLabel,
  normalizePickerDay,
  resolveMonthCount,
  toDraftRange,
} from '@/components/ui/date_range_picker_shared';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import {
  endOfDayLocalValue,
  parseDatetimeLocalValue,
  startOfDayLocalValue,
} from '@/lib/datetime_range';
import { resolvePopoverAlign, resolvePopoverSide } from '@/lib/popover_align';
import { cn } from '@/lib/utils';

export type DateRangePickerProps = {
  id: string;
  label: string;
  from: string;
  to: string;
  onChange: (from: string, to: string) => void;
  disabled?: boolean;
  className?: string;
  labelClassName?: string;
  variant?: 'default' | 'admin';
};

export function DateRangePicker({
  id,
  label,
  from,
  to,
  onChange,
  disabled = false,
  className,
  labelClassName,
  variant = 'default',
}: DateRangePickerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [side, setSide] = useState<'top' | 'bottom'>('bottom');
  const [monthCount, setMonthCount] = useState(resolveMonthCount);
  const [draftRange, setDraftRange] = useState<DateRange | undefined>();

  const fromDate = useMemo(() => parseDatetimeLocalValue(from), [from]);
  const toDate = useMemo(() => parseDatetimeLocalValue(to), [to]);
  const displayLabel = formatRangeLabel(fromDate, toDate);
  const isAdmin = variant === 'admin';
  const calendarVariant = isAdmin ? 'admin' : 'default';

  function resetDraft() {
    setDraftRange(toDraftRange(fromDate, toDate));
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      setMonthCount(resolveMonthCount());
      const trigger = triggerRef.current;
      setSide(resolvePopoverSide(trigger, 320));
      setAlign(resolvePopoverAlign(trigger, undefined, 320));
      resetDraft();
    }
    setOpen(nextOpen);
  }

  function commitDraftRange(range: DateRange | undefined) {
    if (!range?.from || !range.to) {
      return;
    }
    const orderedFrom = range.from <= range.to ? range.from : range.to;
    const orderedTo = range.from <= range.to ? range.to : range.from;
    onChange(startOfDayLocalValue(orderedFrom), endOfDayLocalValue(orderedTo));
    setOpen(false);
  }

  function handleDraftSelect(next: DateRange | undefined) {
    setDraftRange(next);
    commitDraftRange(next);
  }

  const trigger = (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          ref={triggerRef}
          id={id}
          type="button"
          variant="outline"
          disabled={disabled}
          className={cn(
            adminChrome.control,
            isAdmin
              ? 'relative inline-flex w-full max-w-full items-center justify-between gap-2 font-normal'
              : 'flex w-full items-center justify-between gap-2 font-normal',
            !fromDate && 'text-muted-foreground'
          )}
        >
          <CalendarIcon className={cn('h-4 w-4 shrink-0', isAdmin && 'opacity-60')} aria-hidden />
          <span className={cn('whitespace-nowrap', isAdmin && 'min-w-0 flex-1 text-left')}>
            {displayLabel}
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align={align} className="w-auto p-0" side={side}>
        <div className="p-3">
          <Calendar
            mode="range"
            numberOfMonths={monthCount}
            selected={draftRange}
            defaultMonth={draftRange?.from ?? fromDate ?? new Date()}
            variant={calendarVariant}
            onSelect={handleDraftSelect}
          />
        </div>
        <div
          className={
            isAdmin
              ? 'flex justify-end gap-2 border-t border-border p-2'
              : 'flex justify-end gap-2 border-t border-border/50 px-3 py-3'
          }
        >
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              onChange('', '');
              setOpen(false);
            }}
          >
            Clear
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );

  if (isAdmin) {
    return (
      <label className={cn(adminKit.fieldLabelClass, className)}>
        <span className={labelClassName}>{label}</span>
        {trigger}
      </label>
    );
  }

  return (
    <div className={cn('grid w-full min-w-0', adminKit.fieldLabelGap, className)}>
      <Label className={labelClassName} htmlFor={id}>
        {label}
      </Label>
      {trigger}
    </div>
  );
}
