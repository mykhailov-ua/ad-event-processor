/**
 * Canonical spacing and typography for Control Plane admin UI.
 * Domains must not invent gap/p/text scale; use adminKit, adminSpacing, or shell layout exports.
 */

/** Vertical and horizontal gap between siblings in one semantic group. */
export const adminSpacing = {
  gap: {
    /** 4px -- icon + label inside one control. */
    xs: 'gap-1',
    /** 6px -- chip inner, tight label row. */
    sm: 'gap-1.5',
    /** 8px -- button groups, header action clusters, toolbar peers. */
    md: 'gap-2',
    /** 12px -- section stacks, panel interiors, page title block to description. */
    lg: 'gap-3',
    /** 16px -- workspace bands, filter field matrix, main + aside grid. */
    xl: 'gap-4',
  },
  gapX: {
    /** Campaign filter row horizontal cell spacing. */
    filterRow: 'gap-x-3',
    formColumns: 'gap-x-4',
  },
  gapY: {
    filterRow: 'gap-y-4',
    formColumns: 'gap-y-4',
  },
  inset: {
    /** Route canvas padding (PageCanvasInset). */
    canvas: 'px-6 py-4',
    /** Card / selection panel / section panel interior. */
    panel: 'p-4',
    /** Compact band (dialog footer/header sm, dev banners). */
    band: 'px-4 py-2',
    bandCompact: 'px-3 py-2',
    bandLg: 'px-4 py-3',
    /** Table cell horizontal padding. */
    tableCellX: 'px-4',
    /** Page footer horizontal + top padding. */
    footer: 'px-6 pt-3',
    /** Section stack divider top padding. */
    sectionTop: 'pt-3',
    /** Empty state vertical breathing room. */
    emptyState: 'px-6 py-16',
    /** Sidebar nav item horizontal inset. */
    navItemX: 'px-2.5',
    /** Sidebar nav link vertical inset. */
    navItemY: 'py-1',
    /** Sidebar group label inset. */
    navGroupLabel: 'px-2.5 py-1.5',
    /** Sidebar scroll container horizontal inset. */
    sidebarX: 'px-2.5',
    /** App header horizontal inset. */
    headerX: 'px-3 md:px-4',
    /** Combobox / listbox message rows. */
    listMessage: 'px-3 py-2',
    /** Combobox section heading inset. */
    listHeading: 'px-3 py-1',
    /** Combobox section vertical band. */
    listSectionY: 'py-1',
  },
  stack: {
    /** Title + description lines in page header. */
    titleBlock: 'grid gap-1',
  },
  flex: {
    buttonGroup: 'flex flex-wrap items-center gap-2',
    actionsRowEnd: 'flex flex-wrap items-center justify-end gap-2',
    headerActions: 'flex shrink-0 flex-wrap items-center gap-2',
    pageHeader: 'flex flex-wrap items-start justify-between gap-3',
    columnMd: 'flex flex-col gap-2',
    columnLg: 'flex flex-col gap-3',
    workspaceFlat: 'flex min-h-0 flex-1 flex-col gap-4',
    sectionStack:
      'flex flex-col gap-3 border-t border-border pt-3 first:border-t-0 first:pt-0',
    footer: 'flex shrink-0 flex-wrap items-center gap-3 border-0 border-t border-border bg-transparent',
    /** App header three-zone row (nav | search overlay | account). */
    headerBar: 'relative flex h-full items-center gap-3',
    headerStart: 'relative z-[1] flex min-w-0 items-center',
    headerEnd: 'relative z-[1] flex min-w-0 items-center justify-end gap-2',
    headerSearchOverlay:
      'pointer-events-none absolute inset-x-0 top-0 flex h-12 items-center justify-center px-14 md:px-20',
    filterField: 'grid min-w-0 gap-2',
    statusBanner: 'flex items-center gap-2',
  },
  grid: {
    mainAside: 'grid min-h-0 min-w-0 w-full flex-1 gap-4',
    filterMatrix:
      'grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] md:items-end',
    campaignsFilterRow:
      'grid w-full grid-cols-2 gap-x-3 gap-y-2 sm:grid-cols-3 md:grid-cols-4 md:items-end',
    filterFormStack: 'grid w-full justify-items-start gap-4',
    sectionPanel: 'grid gap-3 rounded-[8px] border border-border bg-card p-4 text-card-foreground',
  },
  /** Ops directory matrix (sticky head, zebra); arbitrary variants live here only. */
  opsDirectoryTable:
    'w-full border-collapse text-[13px] leading-[18px] [&_th]:sticky [&_th]:top-0 [&_th]:z-[2] [&_th]:h-[34px] [&_th]:max-h-[34px] [&_th]:bg-card [&_th]:px-4 [&_th]:py-0 [&_th]:text-[11px] [&_th]:font-semibold [&_th]:uppercase [&_th]:leading-[14px] [&_th]:tracking-normal [&_th]:text-muted-foreground [&_th]:shadow-sm [&_td]:h-[34px] [&_td]:max-h-[34px] [&_td]:px-4 [&_td]:py-0 [&_tbody_tr:nth-child(even)_td]:bg-muted/30 [&_tbody_tr:last-child_td]:border-b-0',
} as const;

/** Text roles. Do not use text-sm / text-base / arbitrary px sizes in domains. */
export const adminTypography = {
  pageTitle: 'text-lg font-bold tracking-tight text-foreground',
  sectionTitle: 'text-[13px] font-semibold leading-[18px] text-foreground',
  panelTitle: 'text-[13px] font-semibold leading-[18px] text-foreground',
  body: 'text-[13px] leading-[18px] text-foreground',
  bodyMuted: 'text-[13px] leading-[18px] text-muted-foreground',
  label: 'text-[13px] font-medium leading-[18px] text-foreground',
  labelMuted: 'text-[13px] leading-[18px] text-muted-foreground',
  caption: 'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
  captionPlain: 'text-[11px] leading-[14px] text-muted-foreground',
  tableHeader: 'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
  tableBody: 'text-[13px] leading-[18px] text-foreground',
  tooltip: 'text-[12px] leading-5',
  badge: 'text-xs leading-4',
  monoData: 'font-mono text-[12px] leading-5',
} as const;

export const pageCanvasInsetClass = adminSpacing.inset.canvas;
export const pageWorkspaceFlatClass = adminSpacing.flex.workspaceFlat;
export const pageSectionStackClass = adminSpacing.flex.sectionStack;
export const pageFooterFlatClass = `${adminSpacing.flex.footer} ${adminSpacing.inset.footer}`;
export const customerTabHeaderClass = `${adminSpacing.inset.sectionTop} ${adminTypography.sectionTitle}`;
export const customerDetailSectionClass = `grid ${adminSpacing.gap.xl}`;
export const customerDetailHeaderClass = adminSpacing.stack.titleBlock;
export const opsControlPanelClass = `grid ${adminSpacing.gap.xl}`;
export const opsFilterFieldClass = adminSpacing.flex.filterField;
