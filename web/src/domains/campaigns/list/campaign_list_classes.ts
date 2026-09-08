import { adminKit } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

export const campaignListTableCardClass = cn(uiSurfaces.tableHost, adminKit.panelRadius);

export const campaignListTableSurfaceClass = 'min-w-0 overflow-x-auto';

export const campaignListTableClass =
  'w-max table-fixed border-separate border-spacing-0 text-[13px] leading-[18px] text-foreground';

export const campaignListTableFullWidthClass =
  'w-full max-w-none table-fixed border-separate border-spacing-0 text-[13px] leading-[18px] text-foreground';

export const campaignListThClass =
  'relative h-[34px] max-h-[34px] overflow-visible whitespace-nowrap border-b border-r border-border bg-admin-table-header p-0 align-middle text-[11px] font-bold uppercase leading-[14px] tracking-normal text-muted-foreground last:border-r-0';

export const campaignListPinnedThClass = 'sticky bg-admin-table-header';

export const campaignListPinnedTdClass = 'sticky';

export const campaignListTdClass =
  'h-[34px] max-h-[34px] overflow-hidden whitespace-nowrap border-b border-r border-border px-4 py-0 align-middle text-[13px] text-foreground last:border-r-0';

export const campaignListTfootTdClass =
  'border-t border-r border-border bg-admin-table-totals font-normal text-foreground last:border-r-0';

export const campaignListNameRowCellClass = 'flex h-[34px] w-full min-w-0 items-center gap-1.5';

export const campaignListNameRowTextClass = 'min-w-0 flex-1';

export const campaignListNameTextClass =
  'block whitespace-nowrap select-text font-semibold text-foreground';

export const campaignListNameRowMenuSlotClass = 'flex w-7 shrink-0 items-center justify-center';

/** Fills the entire `<th>`; drag-over highlight and drop target use this shell. */
export const campaignListHeaderCellClass =
  'absolute inset-0 z-[1] flex min-w-0 items-center justify-between gap-2 px-4 text-left';

/** Dashboard tables: label-only header shell without reorder grip. */
export const campaignListHeaderShellClass =
  'absolute inset-0 z-[1] flex min-w-0 items-center px-4 text-left';

/** Dashboard/analytics tables: full header labels; horizontal scroll when tight. */
export const campaignListDashboardHeaderLabelClass = 'min-w-0 flex-1 whitespace-nowrap text-left';

/** Ellipsis for long text in fixed-layout directory tables (campaigns width-probe, landers % cols). */
export const campaignListEllipsisTextClass =
  'block min-w-0 max-w-full overflow-hidden text-ellipsis whitespace-nowrap';

/** Header label clip inside fixed-layout directory tables; pair with `campaignListHeaderShellClass`. */
export const campaignListHeaderLabelClass =
  'min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-left';

/** Numeric headers and body cells: end-aligned tabular figures. */
export const campaignListHeaderLabelNumClass =
  'min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-right';

export const campaignListHeaderCellNumClass =
  'absolute inset-0 z-[1] flex min-w-0 items-center justify-end gap-2 px-4 text-right';

/** Header labels stay start-aligned; body numeric cells use campaignListCellContentNumClass. */

export const campaignListColDragGripClass =
  'relative z-[2] inline-flex shrink-0 cursor-grab items-center justify-center self-center text-muted-foreground/40 hover:text-muted-foreground active:cursor-grabbing';

/** Numeric body cells: end-aligned tabular figures. */
export const campaignListCellContentNumClass =
  'block min-w-0 max-w-full whitespace-nowrap text-right  ';

/** ID / UUID body row: copy control then text; compact horizontal padding (td uses p-0). */
export const campaignListIdCellInnerClass =
  'flex h-[34px] w-full min-w-0 items-center justify-start gap-1 px-2';

export const campaignListCopyRowClass =
  'flex h-[34px] w-full min-w-0 items-center justify-start gap-1.5';

export const campaignListCopyTextClass =
  'shrink-0 select-text whitespace-nowrap  text-xs font-medium text-foreground';

export const campaignListCopyToolsSlotClass = 'flex w-6 shrink-0 items-center justify-center';

export const campaignListCellContentClass = 'block min-w-0 max-w-full whitespace-nowrap';

export const campaignListStatusCellInnerClass =
  'flex h-[34px] w-full items-center justify-start gap-1 px-4';

export const campaignListSelectCellClass =
  'flex h-[34px] w-full items-center justify-center overflow-visible';

export const campaignListSelectHeaderShellClass =
  'absolute inset-0 z-[1] flex items-center justify-center';

/** Hit target centered on `<th>` right border; reorder uses `[data-col-grip]` only. */
export const campaignListColResizeHandleClass =
  'absolute right-0 top-0 z-[4] h-full w-2 translate-x-1/2 cursor-col-resize touch-none select-none';

export const campaignListFilterFieldClass = 'campaigns-filter-field min-w-0';

export const campaignListFilterLabelClass = adminKit.labelCaps;

export const campaignListArchiveButtonClass =
  'border border-destructive bg-destructive/10 text-destructive shadow-none hover:bg-destructive/20';

export const campaignListColumnsMenuClass = cn(
  'w-[min(100vw-1.5rem,40rem)] overflow-hidden border border-border bg-card shadow-lg',
  adminKit.panelRadius
);

export const campaignListColumnsMenuCheckboxClass = cn(
  'border-border bg-card peer-checked:border-primary peer-checked:bg-primary',
  adminKit.controlRadius
);

export const campaignCountrySelectPopoverClass = 'w-[min(100vw-2rem,16rem)]';

export const campaignCountrySelectTriggerClass = cn(
  adminKit.controlHeight,
  adminKit.controlRadius,
  adminKit.controlText,
  'flex w-full min-w-0 items-center justify-between gap-2 border border-border bg-card px-3 py-1 text-left text-foreground'
);

export const campaignCountrySelectSearchClass =
  'flex items-center gap-2 border-b border-border p-2.5';

export const campaignCountrySelectListClass =
  'ui-scrollbar m-0 flex max-h-60 list-none flex-col gap-1 overflow-y-auto p-1';

export const campaignCountrySelectOptionClass = cn(
  adminKit.controlHeight,
  'flex w-full items-center justify-between gap-2 whitespace-nowrap px-2.5 text-left text-[13px] leading-[18px] text-foreground hover:bg-accent',
  adminKit.controlRadius
);

export const campaignCountrySelectOptionSelectedClass = 'bg-accent text-foreground';

export const campaignMetricsPopoverPanelClass = 'w-[22rem] min-w-[22rem]';

export const campaignCountriesOverflowPopoverPanelClass =
  'w-max min-w-[12rem] max-w-[min(calc(100vw-2rem),16rem)]';

export const campaignOverviewDialogClass = 'w-[calc(100%-2rem)] max-w-xl gap-0 p-0';

export const campaignOverviewDialogPanelClass =
  'flex max-h-[min(88vh,44rem)] flex-col gap-0 overflow-hidden p-0 shadow-xl';

export const campaignOverviewScrollClass =
  'ui-scrollbar grid min-h-0 flex-1 auto-rows-max gap-4 overflow-y-auto px-6 pb-4 pt-5 text-center';

export const campaignOverviewHeaderClass = 'grid gap-3 text-center';

export const campaignOverviewSectionClass = cn(
  shellChrome.sectionPanelClass,
  adminKit.panelRadius,
  'text-center'
);

export const campaignOverviewSectionTitleClass =
  'm-0 text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground';

export const campaignOverviewRowsClass = 'mx-auto grid w-full max-w-md';

export const campaignOverviewRowClass =
  'grid grid-cols-[1fr_auto] items-center gap-4 border-b border-border py-2.5 text-left text-[13px] leading-[18px] last:border-b-0';

export const campaignOverviewMetricGridClass = 'mx-auto grid w-full max-w-md grid-cols-3 gap-2';

export const campaignOverviewMetricCardClass = cn(
  'grid min-h-[4.5rem] auto-rows-min content-center justify-items-center gap-1 border border-border bg-card px-1.5 py-2 text-center',
  adminKit.panelRadius
);

export const campaignOverviewMetricCardAccentClass: Record<1 | 2 | 3, string> = {
  1: 'border-chart-1/35 bg-chart-1/10',
  2: 'border-chart-2/35 bg-chart-2/10',
  3: 'border-chart-3/35 bg-chart-3/10',
};

export const campaignOverviewMetricLabelClass =
  'm-0 text-[10px] font-semibold uppercase leading-[14px] text-muted-foreground';

export const campaignOverviewMetricValueClass = 'm-0  text-xs leading-[16px]';

export const campaignOverviewRatesClass = 'mx-auto grid w-full max-w-md gap-2 text-left';

export const campaignOverviewRateRowClass =
  'grid gap-1 border-b border-border py-2 last:border-b-0';

export const campaignOverviewEmptyBannerClass = cn(
  'm-0 bg-muted px-3 py-2 text-center text-[13px] leading-[18px] text-muted-foreground',
  adminKit.panelRadius
);

export const campaignOverviewFooterClass =
  'grid shrink-0 grid-cols-4 gap-2 border-t border-border bg-muted/30 px-4 py-4 sm:px-6';

export const campaignOverviewFooterButtonClass = cn(
  adminKit.controlHeight,
  adminKit.buttonShell,
  'w-full justify-center px-2 text-[12px] leading-4 font-semibold shadow-none sm:px-3 sm:text-[13px] sm:leading-[18px]',
  adminKit.controlRadius
);

export const campaignOverviewPrimaryButtonClass = cn(
  campaignOverviewFooterButtonClass,
  'border-primary bg-primary text-primary-foreground hover:bg-primary/90'
);

export const campaignOverviewOutlineButtonClass = cn(
  campaignOverviewFooterButtonClass,
  'border-border bg-card text-foreground hover:bg-accent'
);
