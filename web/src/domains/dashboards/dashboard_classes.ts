import { adminKit } from '@/lib/admin_kit';
import { pageWorkspaceFlatClass } from '@/shell/page_layout';
import { cn } from '@/lib/utils';

export const dashboardPageWorkspaceClass = pageWorkspaceFlatClass;

export const dashboardCardClass = cn(
  'min-w-0 overflow-hidden border border-border bg-card',
  adminKit.panelRadius
);

export const dashboardCardHeaderClass =
  'flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3';

export const dashboardCardTitleClass = 'm-0 text-sm font-semibold text-foreground';

export const dashboardCardBodyClass = 'px-5 py-4';

export const dashboardKpiGridClass = 'grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7';

export const dashboardKpiTileClass = cn(
  'grid min-w-0 gap-0.5 border border-border bg-card px-3 py-2.5 text-center sm:px-4 sm:py-3',
  adminKit.panelRadius
);

export const dashboardFilterFieldClass = 'campaigns-filter-field min-w-0';

export const dashboardFilterLabelClass = adminKit.labelCaps;

export const dashboardTableSectionHeaderClass =
  'flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3';

export const dashboardTableSectionSplitClass = 'border-t border-border';
