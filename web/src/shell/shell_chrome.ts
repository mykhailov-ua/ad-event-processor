import { adminSpacing, adminTypography } from '@/lib/admin_spacing';

/**
 * Shell chrome bands (tracker header, card section titles). Equal inset on all sides;
 * not used for page body or table cell padding.
 */
export const shellChrome = {
  /** Tracker top bar: h-11; p-2 matches vertical inset around size-7 controls. */
  trackerHeaderClass: `flex h-11 shrink-0 items-center ${adminSpacing.gap.md} border-b border-border px-2`,
  /** Section/card title row inside bordered panels. */
  sectionHeaderBandClass: `flex items-center justify-between ${adminSpacing.gap.md} border-b border-border ${adminSpacing.inset.band}`,
  /** Dialog or raised panel title row. */
  sectionHeaderBandLgClass: `flex items-center justify-between ${adminSpacing.gap.md} border-b border-border ${adminSpacing.inset.bandLg}`,
  /** Dialog footer band. */
  sectionFooterBandLgClass: `flex flex-wrap items-center justify-end ${adminSpacing.gap.md} border-t border-border ${adminSpacing.inset.bandLg}`,
  /** Caption row above a nested table. */
  tableCaptionBandClass: `${adminSpacing.inset.band} ${adminTypography.caption}`,
  /** Compact meta band (error details, dev banner). */
  compactHeaderBandClass: `${adminSpacing.inset.bandCompact} ${adminTypography.bodyMuted}`,
  /** Bordered panel for ops cards and nested sections (replaces legacy ops-section-card). */
  sectionPanelClass: adminSpacing.grid.sectionPanel,
  /** Raised surface block inside a page section stack. */
  surfaceRaisedClass: `rounded-[8px] border border-border bg-card ${adminSpacing.inset.panel} text-card-foreground shadow-none`,
} as const;
