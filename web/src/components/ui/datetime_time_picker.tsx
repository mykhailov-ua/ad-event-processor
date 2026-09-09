import { useEffect, useState } from 'react';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type DatetimeTimePickerProps = {
  hours: number;
  minutes: number;
  disabled?: boolean;
  hourInputId?: string;
  minuteInputId?: string;
  onHoursChange: (hours: number) => void;
  onMinutesChange: (minutes: number) => void;
};

function clampHour(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(23, Math.max(0, Math.trunc(value)));
}

function clampMinute(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(59, Math.max(0, Math.trunc(value)));
}

function digitsOnly(raw: string): string {
  return raw.replace(/\D/g, '');
}

function commitTimeField(
  raw: string,
  clamp: (value: number) => number,
  onChange: (value: number) => void
): string {
  const trimmed = digitsOnly(raw);
  if (trimmed === '') {
    const next = clamp(0);
    onChange(next);
    return String(next);
  }
  const parsed = Number.parseInt(trimmed, 10);
  const next = clamp(Number.isNaN(parsed) ? 0 : parsed);
  onChange(next);
  return String(next);
}

type TimeFieldProps = {
  id: string;
  label: string;
  value: number;
  maxLength: number;
  disabled?: boolean;
  clamp: (value: number) => number;
  onChange: (value: number) => void;
};

function TimeField({
  id,
  label,
  value,
  maxLength,
  disabled = false,
  clamp,
  onChange,
}: TimeFieldProps) {
  const [draft, setDraft] = useState(String(value));
  const [focused, setFocused] = useState(false);

  useEffect(() => {
    if (!focused) {
      setDraft(String(value));
    }
  }, [focused, value]);

  function commit() {
    setDraft((current) => commitTimeField(current, clamp, onChange));
  }

  return (
    <div className={cn('grid min-w-0', adminKit.fieldLabelGap)}>
      <Label htmlFor={id}>{label}</Label>
      <Input
        className="w-14 px-2 text-center tabular-nums"
        disabled={disabled}
        id={id}
        inputMode="numeric"
        maxLength={maxLength}
        type="text"
        value={draft}
        onBlur={() => {
          setFocused(false);
          commit();
        }}
        onChange={(event) => {
          const next = digitsOnly(event.target.value).slice(0, maxLength);
          setDraft(next);
        }}
        onFocus={() => {
          setFocused(true);
        }}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault();
            commit();
            event.currentTarget.blur();
          }
        }}
      />
    </div>
  );
}

export function DatetimeTimePicker({
  hours,
  minutes,
  disabled = false,
  hourInputId = 'datetime-hour',
  minuteInputId = 'datetime-minute',
  onHoursChange,
  onMinutesChange,
}: DatetimeTimePickerProps) {
  return (
    <div className="flex shrink-0 flex-col gap-3 border-l border-border pl-3">
      <TimeField
        clamp={clampHour}
        disabled={disabled}
        id={hourInputId}
        label="Hour"
        maxLength={2}
        value={hours}
        onChange={onHoursChange}
      />
      <TimeField
        clamp={clampMinute}
        disabled={disabled}
        id={minuteInputId}
        label="Minute"
        maxLength={2}
        value={minutes}
        onChange={onMinutesChange}
      />
    </div>
  );
}
