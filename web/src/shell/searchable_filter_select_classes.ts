import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export const searchableFilterSelectPopoverClass = 'w-[min(100vw-2rem,16rem)]';

export const searchableFilterSelectTriggerClass = cn(
  adminKit.controlHeight,
  adminKit.controlRadius,
  adminKit.controlText,
  'flex w-full min-w-0 items-center justify-between gap-2 border border-border bg-card px-3 py-1 text-left text-foreground'
);

export const searchableFilterSelectSearchClass =
  'flex items-center gap-2 border-b border-border p-2.5';

export const searchableFilterSelectListClass =
  'ui-scrollbar m-0 flex max-h-60 list-none flex-col gap-1 overflow-y-auto p-1';

export const searchableFilterSelectOptionClass = cn(
  adminKit.controlHeight,
  'flex w-full items-center justify-between gap-2 whitespace-nowrap px-2.5 text-left text-[13px] leading-[18px] text-foreground hover:bg-accent',
  adminKit.controlRadius
);

export const searchableFilterSelectOptionSelectedClass = 'bg-accent text-foreground';
