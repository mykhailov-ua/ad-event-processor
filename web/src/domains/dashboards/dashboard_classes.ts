import { adminKit } from '@/lib/admin_kit';

export const dashboardPageWorkspaceClass =
  'flex min-h-0 flex-1 flex-col gap-3 border-0 bg-transparent p-0 dark:bg-transparent';

export const dashboardCardClass =
  'min-w-0 overflow-hidden rounded-[10px] border border-border bg-card';

export const dashboardCardHeaderClass =
  'flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3';

export const dashboardCardTitleClass = 'm-0 text-sm font-normal text-foreground';

export const dashboardCardBodyClass = 'px-5 py-4';

export const dashboardKpiGridClass =
  'grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7';

export const dashboardKpiTileClass =
  'grid min-w-0 gap-0.5 rounded-[10px] border border-border bg-card px-3 py-2.5 text-center sm:px-4 sm:py-3';

export const dashboardFilterFieldClass = 'campaigns-filter-field min-w-0';

export const dashboardFilterLabelClass = adminKit.labelCaps;

export const dashboardTableSectionHeaderClass =
  'flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3';

export const dashboardTableSectionSplitClass = 'border-t border-border';
