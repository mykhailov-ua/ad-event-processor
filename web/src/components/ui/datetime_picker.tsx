import { useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import { CalendarIcon } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { adminChrome } from '@/lib/admin_chrome';
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

type CalendarPickerProps = {
  id?: string;
  label?: string;
  value: string;
  onChange: (value: string) => void;
  mode: CalendarPickerMode;
  disabled?: boolean;
  className?: string;
  placeholder?: string;
};

const PLACEHOLDER: Record<CalendarPickerMode, string> = {
  datetime: 'Pick date and time',
  date: 'Select date',
  month: 'Select month',
};

const POPOVER_HEIGHT: Record<CalendarPickerMode, number> = {
  datetime: 440,
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

function CalendarPicker({
  id,
  label,
  value,
  onChange,
  mode,
  disabled = false,
  className,
  placeholder,
}: CalendarPickerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [side, setSide] = useState<'top' | 'bottom'>('bottom');
  const selected = useMemo(() => parsePickerValue(mode, value), [mode, value]);
  const timeValue = useMemo(() => {
    if (mode !== 'datetime' || !selected) {
      return '00:00';
    }
    const pad = (part: number) => String(part).padStart(2, '0');
    return `${pad(selected.getHours())}:${pad(selected.getMinutes())}`;
  }, [mode, selected]);

  const displayLabel = selected
    ? formatPickerDisplay(mode, selected)
    : (placeholder ?? PLACEHOLDER[mode]);

  function applyDate(date: Date | undefined) {
    if (!date) {
      onChange('');
      return;
    }
    if (mode === 'datetime') {
      const [hours, minutes] = timeValue.split(':').map((part) => Number(part));
      onChange(mergeDatetimeLocalDate(date, hours || 0, minutes || 0));
      return;
    }
    onChange(formatPickerValue(mode, date));
    setOpen(false);
  }

  function applyTime(nextTime: string) {
    const base = selected ?? new Date();
    const [hours, minutes] = nextTime.split(':').map((part) => Number(part));
    if (Number.isNaN(hours) || Number.isNaN(minutes)) {
      return;
    }
    onChange(mergeDatetimeLocalDate(base, hours, minutes));
  }

  const field = (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          const trigger = triggerRef.current;
          setSide(resolvePopoverSide(trigger, POPOVER_HEIGHT[mode]));
          setAlign(resolvePopoverAlign(trigger, undefined, 320));
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
          className={cn(
            adminChrome.control,
            'w-full justify-start gap-2 text-left font-normal',
            !selected && 'text-muted-foreground',
            className
          )}
        >
          <CalendarIcon className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
          <span className="min-w-0 truncate">{displayLabel}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align={align} side={side} panelScroll="none">
        <div className="flex justify-center p-3">
          <Calendar
            mode="single"
            selected={selected}
            defaultMonth={selected ?? new Date()}
            onSelect={applyDate}
            autoFocus
          />
        </div>
        {mode === 'datetime' ? (
          <div className="grid gap-2 border-t p-3">
            <Label className="text-xs text-muted-foreground" htmlFor={`${id}-time`}>
              Time
            </Label>
            <Input
              id={`${id}-time`}
              type="time"
              value={timeValue}
              onChange={(event) => applyTime(event.target.value)}
            />
          </div>
        ) : null}
        <div className="flex justify-end gap-2 border-t border-border p-2">
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
        </div>
      </PopoverContent>
    </Popover>
  );

  if (!label) {
    return field;
  }

  return (
    <div className={cn('grid gap-2', className)}>
      <Label htmlFor={id}>{label}</Label>
      {field}
    </div>
  );
}

export type DatetimePickerProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
};

export function DatetimePicker(props: DatetimePickerProps) {
  return <CalendarPicker {...props} mode="datetime" />;
}

export type DatePickerProps = Omit<CalendarPickerProps, 'mode' | 'label'> & {
  label?: string;
};

export function DatePicker(props: DatePickerProps) {
  return <CalendarPicker {...props} mode="date" />;
}

export type MonthPickerProps = Omit<CalendarPickerProps, 'mode' | 'label'> & {
  label?: string;
};

export function MonthPicker(props: MonthPickerProps) {
  return <CalendarPicker {...props} mode="month" />;
}
