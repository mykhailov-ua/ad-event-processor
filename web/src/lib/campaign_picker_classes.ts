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

export const campaignDateRangeClearButtonClass =
  'inline-flex min-h-8 items-center justify-center rounded-md border border-border bg-background px-3 text-[13px] font-medium leading-[18px] text-foreground hover:bg-accent';

export const campaignDateRangeApplyButtonClass =
  'inline-flex min-h-8 items-center justify-center rounded-md border border-primary bg-primary px-3 text-[13px] font-medium leading-[18px] text-primary-foreground hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50';
