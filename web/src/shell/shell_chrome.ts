/**
 * Shell chrome bands (tracker header, card section titles). Equal inset on all sides;
 * not used for page body or table cell padding.
 */
export const shellChrome = {
  /** Tracker top bar: h-11; p-2 matches vertical inset around size-7 controls. */
  trackerHeaderClass:
    'grid h-11 shrink-0 grid-cols-[minmax(0,1fr)_minmax(12rem,28rem)_minmax(0,1fr)] items-center gap-3 border-b border-border bg-card p-2 text-card-foreground',
  /** Section/card title row inside bordered panels. */
  sectionHeaderBandClass: 'grid grid-cols-[1fr_auto] items-center gap-2 border-b border-border p-3',
  /** Dialog or raised panel title row. */
  sectionHeaderBandLgClass: 'shrink-0 border-b border-border p-4 text-left',
  /** Dialog footer band. */
  sectionFooterBandLgClass:
    'shrink-0 grid grid-cols-[1fr_auto] items-center gap-3 border-t border-border bg-muted/20 p-4',
  /** Caption row above a nested table. */
  tableCaptionBandClass:
    'border border-b-0 border-border p-2 text-[13px] font-medium leading-[18px]',
  /** Compact meta band (error details, dev banner). */
  compactHeaderBandClass: 'p-2',
  /** Bordered panel for ops cards and nested sections (replaces legacy ops-section-card). */
  sectionPanelClass: 'grid gap-3 border border-border bg-card p-3',
  /** Raised surface block inside a page section stack. */
  surfaceRaisedClass: 'ui-surface-raised grid gap-3 p-4',
} as const;
