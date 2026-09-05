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
import {
  campaignDateRangeClearButtonClass,
  campaignDateRangeFooterClass,
  campaignDateRangePopoverClass,
  campaignDateRangeTriggerClass,
} from '@/lib/campaign_picker_classes';
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
  variant?: 'default' | 'admin' | 'campaigns';
};

function formatFooterRange(from: Date | undefined, to: Date | undefined): string {
  if (!from) {
    return '';
  }
  if (!to) {
    return format(from, 'MMM d, yyyy');
  }
  return `${format(from, 'MMM d, yyyy')} - ${format(to, 'MMM d, yyyy')}`;
}

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

function estimateCampaignDateRangePopoverWidth(monthCount: number): number {
  return monthCount * 272 + 48;
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
  const isCampaigns = variant === 'campaigns';
  const isStyledPicker = variant === 'admin' || isCampaigns;
  const calendarVariant = isCampaigns ? 'campaigns' : variant === 'admin' ? 'admin' : 'default';
  const draftFooterLabel = formatFooterRange(draftRange?.from, draftRange?.to);

  function resetDraft() {
    setDraftRange(toDraftRange(fromDate, toDate));
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      const nextMonthCount = resolveMonthCount();
      setMonthCount(nextMonthCount);
      const popoverWidth = isCampaigns
        ? estimateCampaignDateRangePopoverWidth(nextMonthCount)
        : 320;
      const trigger = triggerRef.current;
      if (trigger) {
        const rect = trigger.getBoundingClientRect();
        const estimatedHeight = isCampaigns ? 360 : 320;
        const spaceBelow = window.innerHeight - rect.bottom;
        const spaceAbove = rect.top;
        setSide(
          spaceBelow < estimatedHeight && spaceAbove > spaceBelow ? 'top' : 'bottom',
        );
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

  const trigger = (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        {variant === 'admin' || isCampaigns ? (
          <Button
            ref={triggerRef}
            id={id}
            className={cn(
              isCampaigns
                ? cn(campaignDateRangeTriggerClass, 'h-auto justify-between font-normal shadow-none')
                : 'relative inline-flex min-h-7 w-full max-w-full items-center justify-between gap-2 rounded-[5px] border border-border bg-background px-2 py-1 text-[13px] leading-[18px] text-foreground',
              !fromDate && 'text-muted-foreground',
            )}
            disabled={disabled}
            type="button"
            variant="outline"
          >
            <CalendarIcon className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
            <span className="min-w-0 flex-1 truncate text-left">{displayLabel}</span>
          </Button>
        ) : (
          <Button
            ref={triggerRef}
            id={id}
            type="button"
            variant="outline"
            disabled={disabled}
            className={cn(
              'flex min-h-7 w-full items-center justify-between rounded-[5px] border border-border bg-background px-2 py-1 text-[13px] font-normal leading-[18px] text-foreground',
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
          'w-auto p-0',
          isCampaigns && campaignDateRangePopoverClass,
        )}
        panelScroll={isCampaigns ? 'none' : undefined}
        side={side}
      >
        <div className={cn(isCampaigns ? 'p-3 pb-2' : 'p-3')}>
          <Calendar
            mode="range"
            numberOfMonths={monthCount}
            selected={draftRange}
            defaultMonth={draftRange?.from ?? fromDate ?? new Date()}
            variant={calendarVariant}
            onSelect={handleDraftSelect}
          />
        </div>
        {isCampaigns ? (
          <div className={campaignDateRangeFooterClass}>
            <span className="min-w-0 truncate text-[13px] leading-[18px] text-muted-foreground">
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
        ) : (
          <div
            className={
              variant === 'admin'
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
        )}
      </PopoverContent>
    </Popover>
  );

  if (variant === 'admin') {
    return (
      <label className={cn('text-sm font-medium text-foreground', className)}>
        <span className={labelClassName}>{label}</span>
        {trigger}
      </label>
    );
  }

  return (
    <div className={cn('grid w-full min-w-0 gap-1.5', className)}>
      <Label className={labelClassName} htmlFor={id}>{label}</Label>
      {trigger}
    </div>
  );
}
