import { useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import { CalendarIcon } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  mergeDatetimeLocalDate,
  parseDatetimeLocalValue,
} from '@/lib/datetime_range';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { cn } from '@/lib/utils';

export type DatetimePickerProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
};

export function DatetimePicker({
  id,
  label,
  value,
  onChange,
  disabled = false,
  className,
}: DatetimePickerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const selected = useMemo(() => parseDatetimeLocalValue(value), [value]);
  const timeValue = useMemo(() => {
    if (!selected) {
      return '00:00';
    }
    const pad = (part: number) => String(part).padStart(2, '0');
    return `${pad(selected.getHours())}:${pad(selected.getMinutes())}`;
  }, [selected]);

  const displayLabel = selected ? format(selected, 'MMM d, yyyy HH:mm') : 'Pick date and time';

  function applyDate(date: Date | undefined) {
    if (!date) {
      onChange('');
      return;
    }
    const [hours, minutes] = timeValue.split(':').map((part) => Number(part));
    onChange(mergeDatetimeLocalDate(date, hours || 0, minutes || 0));
  }

  function applyTime(nextTime: string) {
    const base = selected ?? new Date();
    const [hours, minutes] = nextTime.split(':').map((part) => Number(part));
    if (Number.isNaN(hours) || Number.isNaN(minutes)) {
      return;
    }
    onChange(mergeDatetimeLocalDate(base, hours, minutes));
  }

  return (
    <div className={cn('grid gap-2', className)}>
      <Label htmlFor={id}>{label}</Label>
      <Popover
        open={open}
        onOpenChange={(nextOpen) => {
          if (nextOpen) {
            setAlign(resolvePopoverAlign(triggerRef.current));
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
              'w-full justify-start text-left font-normal',
              !selected && 'text-muted-foreground',
            )}
          >
            <CalendarIcon className="mr-2 h-4 w-4" />
            {displayLabel}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align={align}>
          <div className="flex justify-center">
            <Calendar mode="single" selected={selected} onSelect={applyDate} autoFocus />
          </div>
          <div className="border-t p-3">
            <Label className="text-xs text-muted-foreground" htmlFor={`${id}-time`}>
              Time
            </Label>
            <Input
              id={`${id}-time`}
              type="time"
              className="mt-1"
              value={timeValue}
              onChange={(event) => applyTime(event.target.value)}
            />
          </div>
          <div className="flex justify-end border-t p-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
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
    </div>
  );
}
