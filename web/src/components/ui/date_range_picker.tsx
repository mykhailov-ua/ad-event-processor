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
  variant?: 'default' | 'admin';
};

export function DateRangePicker({
  id,
  label,
  from,
  to,
  onChange,
  disabled = false,
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
         
        >
          <CalendarIcon  aria-hidden />
          <span >
            {displayLabel}
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align={align} side={side}>
        <div >
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
      <label >
        <span >{label}</span>
        {trigger}
      </label>
    );
  }

  return (
    <div >
      <Label  htmlFor={id}>
        {label}
      </Label>
      {trigger}
    </div>
  );
}
