import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export const campaignListTableCardClass =
  'flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-[10px] border border-border bg-card';

export const campaignListTableSurfaceClass =
  'ui-scrollbar min-h-0 min-w-0 flex-1 overflow-x-auto overflow-y-auto bg-card';

export const campaignListTableClass =
  'w-max min-w-full table-fixed border-separate border-spacing-0 text-[13px] leading-[18px] text-foreground';

export const campaignListThClass =
  'h-[34px] max-h-[34px] overflow-visible whitespace-nowrap border-b border-r border-border bg-muted/50 px-4 py-0 align-middle text-[11px] font-bold uppercase leading-[14px] tracking-normal text-muted-foreground last:border-r-0';

export const campaignListTdClass =
  'h-[34px] max-h-[34px] overflow-hidden whitespace-nowrap border-b border-border bg-inherit px-4 py-0 align-middle text-[13px] text-foreground';

export const campaignListTfootTdClass = 'border-t border-border bg-muted font-semibold text-foreground';

export const campaignListCellToolsClass = 'pr-4';

export const campaignListThNameClass = 'relative pr-9';

export const campaignListTdNameClass = 'relative pr-9';

export const campaignListHeaderCellClass =
  'relative flex h-[34px] min-w-0 items-center gap-0 pr-2';

export const campaignListHeaderLabelClass = 'min-w-0 flex-1 whitespace-nowrap';

export const campaignListHeaderLabelNumClass = 'min-w-0 flex-1 whitespace-nowrap text-right';

export const campaignListCellContentClass = 'block min-w-0 max-w-full whitespace-nowrap';

export const campaignListCellContentNumClass =
  'block min-w-0 max-w-full whitespace-nowrap text-right tabular-nums';

export const campaignListHeaderToolsClass = 'flex w-4 shrink-0 items-stretch justify-end';

export const campaignListBodyToolsGutterClass =
  'pointer-events-none flex w-4 shrink-0 items-stretch justify-end';

export const campaignListSelectCellClass =
  'relative z-[2] flex h-[34px] items-center justify-center overflow-visible';

export const campaignListColDragGripClass =
  'flex h-full w-4 shrink-0 cursor-grab items-center justify-center text-muted-foreground/40 hover:text-muted-foreground active:cursor-grabbing';

export const campaignListColGripClass =
  'absolute right-0 top-0 z-[3] flex h-full w-3 shrink-0 cursor-col-resize touch-none items-center justify-center text-muted-foreground/30 hover:text-muted-foreground';

export const campaignListNumClass = 'text-right';

export const campaignListFilterFieldClass = 'campaigns-filter-field min-w-0';

export const campaignListFilterLabelClass = adminKit.labelCaps;

export const campaignListArchiveButtonClass =
  'border-destructive bg-destructive/10 text-destructive shadow-none hover:bg-destructive/20';

export const campaignListColumnsMenuClass =
  'w-[26rem] overflow-hidden rounded-lg border border-border bg-card shadow-lg';

export const campaignListColumnsMenuCheckboxClass =
  'rounded-[4px] border-border bg-background peer-checked:border-primary peer-checked:bg-primary';

export const campaignCountrySelectPopoverClass = 'w-[min(100vw-2rem,16rem)]';

export const campaignCountrySelectTriggerClass = cn(
  adminKit.controlHeight,
  adminKit.controlRadius,
  adminKit.controlText,
  'flex w-full min-w-0 items-center justify-between gap-2 border border-border bg-background px-2 py-1 text-left text-foreground',
);

export const campaignCountrySelectSearchClass =
  'flex items-center gap-2 border-b border-border px-3 py-2.5';

export const campaignCountrySelectListClass =
  'ui-scrollbar m-0 max-h-60 list-none overflow-y-auto p-1';

export const campaignCountrySelectOptionClass =
  'flex min-h-9 w-full items-center justify-between gap-2 whitespace-nowrap rounded-md px-2.5 py-2 text-left text-[13px] leading-[18px] text-foreground hover:bg-accent';

export const campaignCountrySelectOptionSelectedClass = 'bg-accent text-foreground';

export const campaignMetricsPopoverClass = '[&>div]:w-[22rem] [&>div]:min-w-[22rem]';

export const campaignOverviewDialogClass = cn(
  'w-[calc(100%-2rem)] max-w-md gap-0 p-0',
  '[&>div]:flex [&>div]:max-h-[min(88vh,44rem)] [&>div]:flex-col [&>div]:overflow-hidden [&>div]:gap-0 [&>div]:p-0 [&>div]:shadow-xl',
  '[&>div>button[aria-label="Close"]]:z-10 [&>div>button[aria-label="Close"]]:text-muted-foreground [&>div>button[aria-label="Close"]]:hover:text-foreground',
);

export const campaignOverviewScrollClass =
  'ui-scrollbar min-h-0 flex-1 overflow-y-auto px-6 pb-4 pt-5';

export const campaignOverviewHeaderClass = 'mb-4 grid gap-3';

export const campaignOverviewSectionClass =
  'rounded-[10px] border border-border bg-card p-3';

export const campaignOverviewSectionTitleClass =
  'm-0 text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground';

export const campaignOverviewRowsClass = 'grid';

export const campaignOverviewRowClass =
  'flex items-center justify-between gap-4 border-b border-border py-2.5 text-[13px] leading-[18px] last:border-b-0';

export const campaignOverviewMetricCardClass =
  'flex min-h-[4.5rem] flex-col items-center justify-center rounded-[8px] border border-border bg-card px-1.5 py-2 text-center';

export const campaignOverviewMetricLabelClass =
  'm-0 text-[10px] font-semibold uppercase leading-[14px] text-muted-foreground';

export const campaignOverviewMetricValueClass =
  'm-0 text-xs font-bold leading-[16px] tabular-nums text-foreground';

export const campaignOverviewRatesClass = 'grid gap-2';

export const campaignOverviewRateRowClass =
  'grid gap-1 border-b border-border py-2 last:border-b-0';

export const campaignOverviewEmptyBannerClass =
  'm-0 rounded-[8px] bg-muted px-3 py-2 text-center text-[13px] leading-[18px] text-muted-foreground';

export const campaignOverviewFooterClass =
  'grid shrink-0 grid-cols-2 gap-2 border-t border-border bg-muted/30 px-6 py-4';

export const campaignOverviewFooterButtonClass =
  'h-auto min-h-9 w-full justify-center rounded-[5px] px-3 py-2 text-[13px] font-semibold leading-[18px] shadow-none';

export const campaignOverviewPrimaryButtonClass = cn(
  campaignOverviewFooterButtonClass,
  'border-primary bg-primary text-primary-foreground hover:bg-primary/90',
);

export const campaignOverviewOutlineButtonClass = cn(
  campaignOverviewFooterButtonClass,
  'border-border bg-background text-foreground hover:bg-accent',
);
