import { useMemo, useRef, useState } from 'react';
import { CalendarIcon } from 'lucide-react';
import type { DateRange } from 'react-day-picker';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import {
  estimateToolbarDateRangePopoverWidth,
  formatFooterRange,
  formatRangeLabel,
  normalizePickerDay,
  resolveMonthCount,
  toDraftRange,
} from '@/components/ui/date_range_picker_shared';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  endOfDayLocalValue,
  parseDatetimeLocalValue,
  startOfDayLocalValue,
} from '@/lib/datetime_range';
import {
  campaignDateRangeClearButtonClass,
  campaignDateRangeFooterClass,
  campaignDateRangePopoverClass,
  campaignDateRangeTriggerClass,
} from '@/lib/campaign_picker_classes';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type ToolbarDateRangePickerProps = {
  id: string;
  label: string;
  from: string;
  to: string;
  onChange: (from: string, to: string) => void;
  disabled?: boolean;
  className?: string;
  labelClassName?: string;
};

export function ToolbarDateRangePicker({
  id,
  label,
  from,
  to,
  onChange,
  disabled = false,
  className,
  labelClassName,
}: ToolbarDateRangePickerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [side, setSide] = useState<'top' | 'bottom'>('bottom');
  const [monthCount, setMonthCount] = useState(resolveMonthCount);
  const [draftRange, setDraftRange] = useState<DateRange | undefined>();

  const fromDate = useMemo(() => parseDatetimeLocalValue(from), [from]);
  const toDate = useMemo(() => parseDatetimeLocalValue(to), [to]);
  const displayLabel = formatRangeLabel(fromDate, toDate);
  const draftFooterLabel = formatFooterRange(draftRange?.from, draftRange?.to);

  function resetDraft() {
    setDraftRange(toDraftRange(fromDate, toDate));
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      const nextMonthCount = resolveMonthCount();
      setMonthCount(nextMonthCount);
      const popoverWidth = estimateToolbarDateRangePopoverWidth(nextMonthCount);
      const trigger = triggerRef.current;
      if (trigger) {
        const rect = trigger.getBoundingClientRect();
        const estimatedHeight = 360;
        const spaceBelow = window.innerHeight - rect.bottom;
        const spaceAbove = rect.top;
        setSide(spaceBelow < estimatedHeight && spaceAbove > spaceBelow ? 'top' : 'bottom');
      } else {
        setSide('bottom');
      }
      setAlign(resolvePopoverAlign(trigger, undefined, popoverWidth));
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

  return (
    <div className={cn('grid w-full min-w-0', adminKit.fieldLabelGap, className)}>
      <Label className={labelClassName} htmlFor={id}>
        {label}
      </Label>
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <Button
            ref={triggerRef}
            id={id}
            className={cn(campaignDateRangeTriggerClass, !fromDate && 'text-muted-foreground')}
            disabled={disabled}
            type="button"
            variant="outline"
          >
            <CalendarIcon className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
            <span className="min-w-0 flex-1 whitespace-nowrap text-left">{displayLabel}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align={align}
          className={cn('w-auto p-0', campaignDateRangePopoverClass)}
          panelScroll="none"
          side={side}
        >
          <div className="p-3 pb-2">
            <Calendar
              mode="range"
              numberOfMonths={monthCount}
              selected={draftRange}
              defaultMonth={draftRange?.from ?? fromDate ?? new Date()}
              variant="toolbar"
              onSelect={handleDraftSelect}
            />
          </div>
          <div className={campaignDateRangeFooterClass}>
            <span className="min-w-0 whitespace-nowrap text-[13px] leading-[18px] text-muted-foreground">
              {draftFooterLabel || 'Pick date range'}
            </span>
            <Button
              className={campaignDateRangeClearButtonClass}
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
    </div>
  );
}
