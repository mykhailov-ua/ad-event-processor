import { adminKit } from '@/lib/admin_kit';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

export const dashboardPrefsDialogHeaderClass = shellChrome.sectionHeaderBandLgClass;

export const dashboardPrefsDialogTitleClass = 'm-0 text-base font-semibold leading-6 text-primary';

export const dashboardPrefsDialogSectionClass = 'grid gap-3';

export const dashboardPrefsDialogSectionTitleClass =
  'm-0 text-[13px] font-semibold leading-[18px] text-foreground';

export const dashboardPrefsDialogSectionBodyClass = 'grid gap-4';

export const dashboardPrefsFieldClass = 'grid gap-2';

export const dashboardPrefsFieldLabelClass =
  'm-0 text-[13px] font-medium leading-[18px] text-muted-foreground';

export const dashboardPrefsChipBoxClass = cn(
  'flex min-h-7 flex-wrap items-center gap-1.5 border border-border bg-background px-2.5 py-1',
  adminKit.panelRadius
);

export const dashboardPrefsChipEmptyClass = 'text-[13px] leading-[18px] text-muted-foreground';

export const dashboardPrefsChipClass = cn(
  'inline-flex max-w-full items-center gap-1 border border-border bg-muted/60 px-2 py-0.5 text-[12px] leading-4 text-foreground',
  adminKit.pillRadius
);

export const dashboardPrefsChipLabelClass = 'whitespace-nowrap';

export const dashboardPrefsChipRemoveClass = cn(
  'inline-flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground',
  adminKit.pillRadius
);

export const dashboardPrefsColumnSummaryClass = cn(
  'm-0 border border-border bg-background px-3 py-2 text-[13px] leading-[18px] text-foreground',
  adminKit.panelRadius
);

export const dashboardPrefsCheckboxListClass = cn(
  'ui-scrollbar overflow-y-auto border border-border bg-background px-3 py-2',
  adminKit.panelRadius
);

export const dashboardPrefsCheckboxRowClass = cn(
  'flex cursor-pointer items-center gap-2.5 px-1 py-1.5 hover:bg-muted/50',
  adminKit.controlRadius
);

export const dashboardPrefsCheckboxLabelClass = 'text-[13px] leading-[18px] text-foreground';

export const dashboardPrefsDialogFooterClass = shellChrome.sectionFooterBandLgClass;

export const dashboardPrefsRestoreClass =
  'text-[13px] font-medium leading-[18px] text-admin-positive transition-colors hover:text-primary';

export const dashboardPrefsFooterActionsClass = 'flex flex-wrap items-center gap-2';
