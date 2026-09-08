import { adminKit } from '@/lib/admin_kit';
import { ADMIN_TABULAR_CLASS } from '@/lib/admin_typography';
import { pageWorkspaceFlatClass } from '@/shell/page_layout';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

export const settingsPageWorkspaceClass = pageWorkspaceFlatClass;

export const settingsCardClass = cn(
  'min-w-0 overflow-hidden border border-border bg-card',
  adminKit.panelRadius
);

export const settingsCardHeaderClass = shellChrome.sectionHeaderBandClass;

export const settingsCardTitleClass = 'm-0 text-[13px] font-semibold leading-[18px] text-foreground';

export const settingsCardBodyClass = 'px-5 py-4';

export const settingsSectionTitleClass = adminKit.labelCaps;

export const settingsBentoTypeClass = cn(
  'font-sans',
  ADMIN_TABULAR_CLASS,
  'text-[13px] leading-[18px] font-normal'
);

export const settingsRowClass =
  'grid min-h-[34px] grid-cols-[1fr_auto] items-center gap-4 border-b border-border px-3 py-2 text-[13px] leading-[18px] last:border-b-0';

export const settingsGridCellInnerClass =
  'grid min-h-[34px] grid-cols-[minmax(7.5rem,46%)_minmax(0,1fr)] items-center gap-3';

export const settingsBentoTableClass = cn(
  'hidden w-full table-fixed border-collapse xl:table',
  settingsBentoTypeClass,
  adminKit.panelRadius,
  'overflow-hidden border border-border bg-muted/10'
);

export const settingsBentoHeaderCellClass = cn(
  settingsBentoTypeClass,
  'border-b border-r border-border px-3 py-2 text-left align-middle text-[11px] uppercase leading-[14px] text-muted-foreground last:border-r-0'
);

export const settingsBentoBodyCellClass =
  'min-h-[34px] border-b border-r border-border px-3 py-2 align-middle last:border-r-0';

export const settingsBentoMobileStackClass = cn('grid gap-5 xl:hidden', settingsBentoTypeClass);

export const settingsRowLabelClass = 'min-w-0 text-muted-foreground';

export const settingsRowValueClass =
  'flex min-w-0 items-center justify-end gap-2 text-right text-foreground';

export const settingsColumnClass = 'min-w-0 grid gap-2.5';

export const settingsColumnPanelClass = cn(
  'min-w-0 divide-y divide-border overflow-hidden border border-border bg-muted/10',
  adminKit.panelRadius
);

export const settingsCollapsibleSummaryClass =
  `grid cursor-pointer list-none grid-cols-[1fr_auto] items-center gap-2 ${shellChrome.compactHeaderBandClass} marker:content-none [&::-webkit-details-marker]:hidden`;

export const settingsCollapsibleBodyClass = 'border-t border-border px-5 py-4';

export const settingsHintClass = 'm-0 text-[13px] leading-[18px] text-muted-foreground';

export const settingsFormStackClass = 'grid w-full gap-4';
