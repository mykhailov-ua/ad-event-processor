import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { shellChrome } from '@/shell/shell_chrome';

export const DIRECTORY_FILTER_GRID = adminSpacing.grid.filterMatrix;

export const CAMPAIGNS_FILTER_ROW = adminSpacing.grid.campaignsFilterRow;

/** Export hub parity: left-aligned filter band inside panel. */
export const DIRECTORY_CONTENT_BAND_CLASS = 'w-full min-w-0 max-w-[75%]';

export const DIRECTORY_FILTER_FORM_STACK_CLASS = adminSpacing.grid.filterFormStack;

export const DIRECTORY_FIELD_LABEL_CLASS =
  'block min-h-[18px] whitespace-nowrap leading-[18px]';

export const AUTO_FILL_FILTER_GRID = adminSpacing.grid.filterMatrix;

/** Field + single action (Load/Apply): caps band width; action column stays content-sized. */
export const INLINE_FILTER_ACTION_GRID_CLASS =
  'grid gap-4 md:grid-cols-[minmax(12rem,1fr)_auto] md:items-end';

/** Field + single action; wider cap for customer/campaign scope bars. */
export const INLINE_FILTER_ACTION_GRID_WIDE_CLASS =
  'grid gap-4 md:grid-cols-[minmax(16rem,1fr)_auto] md:items-end';

/** Field + two actions (poll / download). */
export const INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS =
  'grid gap-4 md:grid-cols-[minmax(12rem,1fr)_auto] md:items-end';

/** Field + three actions (poll / cancel / download). */
export const INLINE_FILTER_ACTION_GRID_THREE_ACTIONS_CLASS =
  'grid gap-4 md:grid-cols-[minmax(12rem,1fr)_auto] md:items-end';

/** Two fields + one action on a single row. */
export const INLINE_FILTER_ACTION_GRID_TWO_FIELDS_CLASS =
  'grid gap-4 md:grid-cols-[repeat(2,minmax(12rem,1fr))_auto] md:items-end';

/** Stacked edit form on a page: cap the shell, not a field inside a full-width Card. */
export const NARROW_EDIT_FORM_CLASS = 'mx-auto w-full max-w-xl';

/** Wider stacked edit form (textarea / JSON tools). */
export const NARROW_EDIT_FORM_WIDE_CLASS = 'mx-auto w-full max-w-3xl';

/** Small field stack inside ui-filter-panel chrome. */
export const FILTER_PANEL_NARROW_CLASS = 'mx-auto w-full max-w-xl';

/** Wider field stack inside ui-filter-panel chrome. */
export const FILTER_PANEL_WIDE_CLASS = 'mx-auto w-full max-w-3xl';

/** Sparse read-only summary band (team overview, fraud preview). */
export const FILTER_PANEL_SUMMARY_CLASS = adminTypography.bodyMuted;

/** Tighter vertical rhythm inside ui-filter-panel; keeps border, fill, and inset padding. */
export const FILTER_PANEL_FLAT_CLASS = adminSpacing.gap.lg;

/** Campaign editor main column beside paths aside. */
export const EDITOR_MAIN_COLUMN_CLASS = 'min-w-0 flex-1';

/** Inline controls (pagination, export) without stretching buttons to grid tracks. */
export const COMPACT_TOOLBAR_ROW_CLASS = adminSpacing.flex.buttonGroup;

/** Bordered section panel (ops cards, overview blocks). */
export const SECTION_PANEL_CLASS = shellChrome.sectionPanelClass;

/** Raised surface inside page section stack. */
export const SECTION_SURFACE_RAISED_CLASS = shellChrome.surfaceRaisedClass;
