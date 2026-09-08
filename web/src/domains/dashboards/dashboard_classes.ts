import { adminKit } from '@/lib/admin_kit';
import { pageWorkspaceFillClass } from '@/shell/page_layout';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

export const dashboardPageWorkspaceClass = pageWorkspaceFillClass;

export const dashboardCardClass = cn('min-w-0 border border-border bg-card', adminKit.panelRadius);

/** Dashboard table host: horizontal scroll only on this surface (outer card scrolls vertically). */
export const dashboardTableSurfaceClass = 'ui-scrollbar min-w-0 overflow-x-auto bg-card';

export const dashboardTableScrollHostClass = 'ui-scrollbar min-h-0 min-w-0 flex-1 overflow-y-auto';

export const dashboardTableScrollHostBoundedClass =
  'ui-scrollbar max-h-[min(28rem,60vh)] min-w-0 overflow-y-auto';

/** Fixed-layout dashboard tables: width comes from colgroup + inline tableStyle (no w-max). */
export const dashboardTableClass =
  'table-fixed border-collapse text-[13px] leading-[18px] text-foreground';

export const dashboardTableThClass =
  'relative h-[34px] max-h-[34px] overflow-visible whitespace-nowrap border-b border-r border-border bg-admin-table-header p-0 align-middle text-[11px] font-bold uppercase leading-[14px] tracking-normal text-muted-foreground last:border-r-0';

export const dashboardTableTdClass =
  'h-[34px] max-h-[34px] overflow-visible whitespace-nowrap border-b border-r border-border px-4 py-0 align-middle text-[13px] text-foreground last:border-r-0';

export const dashboardTableTfootTdClass =
  'border-t border-r border-border bg-admin-table-totals font-normal text-foreground last:border-r-0';

export const dashboardCardHeaderClass = shellChrome.sectionHeaderBandClass;

export const dashboardCardTitleClass = 'm-0 text-[13px] font-bold leading-[18px] text-foreground';

export const dashboardCardBodyClass = 'px-5 py-4';

export const dashboardKpiGridClass = 'grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7';

export const dashboardKpiTileClass = cn(
  'grid min-w-0 gap-0.5 border border-border bg-card px-3 py-2.5 text-center sm:px-4 sm:py-3',
  adminKit.panelRadius
);

export const dashboardFilterFieldClass = 'campaigns-filter-field min-w-0';

export const dashboardFilterLabelClass = adminKit.labelCaps;

export const dashboardTableSectionHeaderClass = shellChrome.sectionHeaderBandClass;

export const dashboardTableSectionSplitClass = 'border-t border-border';

export const dashboardTablesStackClass = 'grid min-w-0 gap-3';

export const dashboardTablesPairRowClass = 'grid min-w-0 gap-3 lg:grid-cols-2 lg:items-stretch';

export const dashboardTablesPairSlotClass = 'flex h-full min-h-0 min-w-0 flex-col';
