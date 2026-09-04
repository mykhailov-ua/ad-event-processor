import { cn } from '@/lib/utils';

export const dashboardPrefsDialogContentClass = cn(
  'max-w-2xl p-0',
  '[&>div.relative]:flex [&>div.relative]:max-h-[min(88vh,44rem)] [&>div.relative]:flex-col [&>div.relative]:overflow-hidden [&>div.relative]:p-0 [&>div.relative]:shadow-xl',
  "[&>div.relative>button[aria-label='Close']]:z-10",
);

export const dashboardPrefsDialogHeaderClass =
  'shrink-0 border-b border-border px-5 py-4 text-left';

export const dashboardPrefsDialogTitleClass =
  'm-0 text-base font-semibold leading-6 text-primary';

export const dashboardPrefsDialogScrollClass =
  'ui-scrollbar min-h-0 flex-1 overflow-y-auto overscroll-y-contain px-5 py-4';

export const dashboardPrefsDialogSectionClass = 'grid gap-3';

export const dashboardPrefsDialogSectionDividerClass =
  'mt-5 grid gap-3 border-t border-border pt-5';

export const dashboardPrefsDialogSectionTitleClass =
  'm-0 text-sm font-semibold leading-5 text-foreground';

export const dashboardPrefsDialogSectionBodyClass = 'grid gap-4';

export const dashboardPrefsFieldClass = 'grid gap-2';

export const dashboardPrefsFieldLabelClass =
  'm-0 text-[13px] font-medium leading-[18px] text-muted-foreground';

export const dashboardPrefsChipBoxClass =
  'flex min-h-10 flex-wrap items-center gap-1.5 rounded-[8px] border border-border bg-background px-2.5 py-2';

export const dashboardPrefsChipEmptyClass =
  'text-[13px] leading-[18px] text-muted-foreground';

export const dashboardPrefsChipClass =
  'inline-flex max-w-full items-center gap-1 rounded-full border border-border bg-muted/60 px-2 py-0.5 text-[12px] leading-4 text-foreground';

export const dashboardPrefsChipLabelClass = 'truncate';

export const dashboardPrefsChipRemoveClass =
  'inline-flex shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground';

export const dashboardPrefsColumnSummaryClass =
  'm-0 rounded-[8px] border border-border bg-background px-3 py-2 text-[13px] leading-[18px] text-foreground';

export const dashboardPrefsCheckboxListClass =
  'ui-scrollbar overflow-y-auto rounded-[8px] border border-border bg-background px-3 py-2';

export const dashboardPrefsCheckboxRowClass =
  'flex cursor-pointer items-center gap-2.5 rounded-[4px] px-1 py-1.5 hover:bg-muted/50';

export const dashboardPrefsCheckboxLabelClass =
  'text-[13px] leading-[18px] text-foreground';

export const dashboardPrefsDialogFooterClass =
  'shrink-0 flex-row items-center justify-between gap-3 border-t border-border bg-muted/20 px-5 py-4 sm:justify-between';

export const dashboardPrefsRestoreClass =
  'text-[13px] font-medium leading-[18px] text-emerald-600 transition-colors hover:text-emerald-700 dark:text-emerald-400 dark:hover:text-emerald-300';

export const dashboardPrefsFooterActionsClass = 'flex flex-wrap items-center gap-2';
