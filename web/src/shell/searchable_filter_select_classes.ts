import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { uiScrollbarClass } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

export const searchableFilterSelectPopoverClass =
  'w-max min-w-[14rem] max-w-[20rem] p-0';

export const searchableFilterSelectTriggerClass = cn(
  adminChrome.control,
  'inline-flex w-full items-center justify-between gap-2 text-left font-normal'
);

export const searchableFilterSelectSearchRowClass =
  'flex items-center gap-2 border-b border-border p-2';

export const searchableFilterSelectSearchClass = cn(
  adminChrome.controlFieldInset,
  adminKit.nestedRadius,
  'h-8 min-w-0 w-full flex-1 border border-border bg-admin-control px-2 [appearance:textfield] [&::-webkit-search-cancel-button]:hidden [&::-webkit-search-decoration]:hidden'
);

export const searchableFilterSelectListClass = cn(
  'flex flex-col gap-0.5 overflow-x-hidden overflow-y-auto p-1',
  uiScrollbarClass,
  'scrollbar-overlay',
  'max-h-60'
);

export const searchableFilterSelectGroupLabelClass = cn(adminKit.labelCaps, 'px-2 py-1.5');

export const searchableFilterSelectOptionClass = cn(
  adminChrome.menuItem,
  'w-full justify-between gap-2 pr-8 text-left'
);

export const searchableFilterSelectOptionSelectedClass = adminChrome.menuItemSelected;

export const searchableFilterSelectEmptyClass =
  'px-3 py-2 text-[13px] leading-[18px] text-muted-foreground';
