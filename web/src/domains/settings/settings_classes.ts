import { adminKit } from '@/lib/admin_kit';

export const settingsPageWorkspaceClass =
  'flex min-h-0 flex-1 flex-col gap-3 border-0 bg-transparent p-0 dark:bg-transparent';

export const settingsCardClass =
  'min-w-0 overflow-hidden rounded-[10px] border border-border bg-card';

export const settingsCardHeaderClass =
  'flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3';

export const settingsCardTitleClass = 'm-0 text-sm font-normal text-foreground';

export const settingsCardBodyClass = 'px-5 py-4';

export const settingsSectionTitleClass = adminKit.labelCaps;

export const settingsRowClass =
  'flex min-h-[34px] items-center justify-between gap-4 border-b border-border px-3 py-2 text-[13px] leading-[18px] last:border-b-0';

export const settingsRowLabelClass = 'min-w-0 shrink text-muted-foreground';

export const settingsRowValueClass =
  'flex min-w-0 flex-wrap items-center justify-end gap-2 text-foreground';

export const settingsColumnsGridClass = 'grid gap-5 xl:grid-cols-3';

export const settingsColumnClass = 'min-w-0 grid gap-2.5';

export const settingsColumnPanelClass =
  'min-w-0 divide-y divide-border overflow-hidden rounded-[8px] border border-border bg-muted/10';

export const settingsCollapsibleSummaryClass =
  'flex cursor-pointer list-none items-center justify-between gap-2 px-5 py-3 marker:content-none [&::-webkit-details-marker]:hidden';

export const settingsCollapsibleBodyClass = 'border-t border-border px-5 py-4';

export const settingsHintClass = 'm-0 text-[13px] leading-[18px] text-muted-foreground';

export const settingsFormStackClass = 'grid w-full gap-4';
