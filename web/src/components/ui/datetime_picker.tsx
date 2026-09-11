import { useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import { CalendarIcon } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { DatetimeTimePicker } from '@/components/ui/datetime_time_picker';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { adminKit } from '@/lib/admin_kit';
import {
  formatDateValue,
  formatMonthValue,
  mergeDatetimeLocalDate,
  parseDateValue,
  parseDatetimeLocalValue,
  parseMonthValue,
} from '@/lib/datetime_range';
import { resolvePopoverAlign, resolvePopoverSide } from '@/lib/popover_align';
import { cn } from '@/lib/utils';

type CalendarPickerMode = 'datetime' | 'date' | 'month';

function timeSelectionEnabled(mode: CalendarPickerMode, showTime?: boolean): boolean {
  return mode === 'datetime' && (showTime ?? true);
}

export type CalendarPickerProps = {
  id?: string;
  label?: string;
  value: string;
  onChange: (value: string) => void;
  mode: CalendarPickerMode;
  /** When mode is datetime: hour/minute inputs and Apply footer. Default true. */
  showTime?: boolean;
  disabled?: boolean;
  placeholder?: string;
};

const PLACEHOLDER: Record<CalendarPickerMode, string> = {
  datetime: 'Pick date and time',
  date: 'Select date',
  month: 'Select month',
};

const POPOVER_HEIGHT: Record<CalendarPickerMode, number> = {
  datetime: 320,
  date: 360,
  month: 360,
};

function parsePickerValue(mode: CalendarPickerMode, value: string): Date | undefined {
  if (mode === 'datetime') {
    return parseDatetimeLocalValue(value);
  }
  if (mode === 'date') {
    return parseDateValue(value);
  }
  return parseMonthValue(value);
}

function formatPickerValue(mode: CalendarPickerMode, date: Date): string {
  if (mode === 'datetime') {
    return mergeDatetimeLocalDate(date, date.getHours(), date.getMinutes());
  }
  if (mode === 'date') {
    return formatDateValue(date);
  }
  return formatMonthValue(date);
}

function formatPickerDisplay(mode: CalendarPickerMode, date: Date): string {
  if (mode === 'datetime') {
    return format(date, 'MMM d, yyyy HH:mm');
  }
  if (mode === 'date') {
    return format(date, 'MMM d, yyyy');
  }
  return format(date, 'MMMM yyyy');
}

export function CalendarPicker({
  id,
  label,
  value,
  onChange,
  mode,
  showTime,
  disabled = false,
  placeholder,
}: CalendarPickerProps) {
  const timeEnabled = timeSelectionEnabled(mode, showTime);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [side, setSide] = useState<'top' | 'bottom'>('bottom');
  const selected = useMemo(() => parsePickerValue(mode, value), [mode, value]);
  const selectedHours = selected?.getHours() ?? 0;
  const selectedMinutes = selected?.getMinutes() ?? 0;
  const popoverHeight = timeEnabled ? POPOVER_HEIGHT.datetime : POPOVER_HEIGHT.date;

  const displayLabel = selected
    ? timeEnabled
      ? formatPickerDisplay(mode, selected)
      : formatPickerDisplay('date', selected)
    : (placeholder ?? (timeEnabled ? PLACEHOLDER.datetime : PLACEHOLDER.date));

  function applyDate(date: Date | undefined) {
    if (!date) {
      onChange('');
      return;
    }
    if (mode === 'datetime') {
      onChange(mergeDatetimeLocalDate(date, selectedHours, selectedMinutes));
      if (!timeEnabled) {
        setOpen(false);
      }
      return;
    }
    onChange(formatPickerValue(mode, date));
    setOpen(false);
  }

  function applyHours(hours: number) {
    const base = selected ?? new Date();
    onChange(mergeDatetimeLocalDate(base, hours, selectedMinutes));
  }

  function applyMinutes(minutes: number) {
    const base = selected ?? new Date();
    onChange(mergeDatetimeLocalDate(base, selectedHours, minutes));
  }

  const field = (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          const trigger = triggerRef.current;
          setSide(resolvePopoverSide(trigger, popoverHeight));
          setAlign(resolvePopoverAlign(trigger, undefined, timeEnabled ? 380 : 320));
        }
        setOpen(nextOpen);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          ref={triggerRef}
          id={id}
          type="button"
          variant="outline"
          disabled={disabled}
          className="w-full min-w-0 justify-start gap-2 overflow-hidden font-normal"
          title={selected ? displayLabel : undefined}
        >
          <CalendarIcon aria-hidden className="size-4 shrink-0 text-muted-foreground" />
          <span
            className={cn(
              'min-w-0 truncate whitespace-nowrap text-left',
              !selected && 'text-muted-foreground'
            )}
          >
            {displayLabel}
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align={align}
        className="w-auto p-0"
        panelClassName="p-3"
        panelScroll="none"
        side={side}
      >
        <div className={cn('flex flex-col gap-3', timeEnabled && 'sm:flex-row sm:items-start')}>
          <Calendar
            autoFocus
            defaultMonth={selected ?? new Date()}
            mode="single"
            selected={selected}
            onSelect={applyDate}
          />
          {timeEnabled ? (
            <DatetimeTimePicker
              disabled={disabled}
              hourInputId={id ? `${id}-hour` : 'datetime-hour'}
              hours={selectedHours}
              minuteInputId={id ? `${id}-minute` : 'datetime-minute'}
              minutes={selectedMinutes}
              onHoursChange={applyHours}
              onMinutesChange={applyMinutes}
            />
          ) : null}
        </div>
        <div className="mt-3 flex items-center justify-between gap-2 border-t border-border pt-3">
          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              onChange('');
              setOpen(false);
            }}
          >
            Clear
          </Button>
          {timeEnabled ? (
            <Button type="button" variant="brand" onClick={() => setOpen(false)}>
              Apply
            </Button>
          ) : null}
        </div>
      </PopoverContent>
    </Popover>
  );

  if (!label) {
    return field;
  }

  return (
    <div className={cn('grid min-w-0', adminKit.fieldLabelGap)}>
      <Label
        className="block min-h-[18px] truncate whitespace-nowrap leading-[18px]"
        htmlFor={id}
      >
        {label}
      </Label>
      {field}
    </div>
  );
}

export type DatetimePickerProps = {
  id: string;
  label?: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  /** Hour/minute inputs in popover. Default true. Wire value stays YYYY-MM-DDTHH:mm. */
  showTime?: boolean;
};

export function DatetimePicker({ showTime = true, ...props }: DatetimePickerProps) {
  return <CalendarPicker {...props} mode="datetime" showTime={showTime} />;
}

export type DatePickerProps = Omit<CalendarPickerProps, 'mode' | 'label' | 'showTime'> & {
  label?: string;
};

export function DatePicker(props: DatePickerProps) {
  return <CalendarPicker {...props} mode="date" showTime={false} />;
}

export type MonthPickerProps = Omit<CalendarPickerProps, 'mode' | 'label'> & {
  label?: string;
};

export function MonthPicker(props: MonthPickerProps) {
  return <CalendarPicker {...props} mode="month" />;
}
