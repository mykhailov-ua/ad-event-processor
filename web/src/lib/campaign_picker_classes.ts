import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export const campaignDateRangeTriggerClass = cn(
  adminKit.controlHeight,
  adminKit.controlRadius,
  adminKit.controlText,
  'flex w-full min-w-0 items-center justify-between gap-1.5 border border-border bg-background px-2 py-0 font-normal text-foreground shadow-none',
);

export const campaignDateRangePopoverClass = 'text-foreground [&_.group\\/calendar]:[--cell-size:2rem]';

export const campaignDateRangeFooterClass =
  'flex items-center justify-between gap-3 border-t border-border px-4 py-3';

export const campaignDateRangeClearButtonClass = 'shrink-0';
