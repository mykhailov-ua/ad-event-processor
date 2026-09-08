import { shellChrome } from '@/shell/shell_chrome';

// Named Tailwind layout contracts (shared style contracts). Not BEM / not app.css.
// ui-filter-panel (app.css): grid gap-4 stack.
// Matrix filter rows: add grid gap-4 items-end md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))].
// Tighter bands: override gap-2 / gap-3 on the panel element.
// Consumers use DirectoryFilterForm layout=... -- do not copy these strings into domains.
export const DIRECTORY_FILTER_GRID =
  'grid grid-cols-[repeat(auto-fill,11rem)] items-end gap-x-4 gap-y-3';

export const CAMPAIGNS_FILTER_ROW =
  'grid w-full grid-cols-2 items-end gap-x-4 gap-y-3 md:grid-cols-4 xl:grid-cols-7';

export const AUTO_FILL_FILTER_GRID =
  'grid grid-cols-[repeat(auto-fill,12rem)] items-end gap-x-4 gap-y-4';

/** Field + single action (Load/Apply): caps band width; action column stays content-sized. */
export const INLINE_FILTER_ACTION_GRID_CLASS =
  'grid w-full max-w-md grid-cols-[minmax(0,1fr)_auto] items-end gap-4';

/** Field + single action; wider cap for customer/campaign scope bars. */
export const INLINE_FILTER_ACTION_GRID_WIDE_CLASS =
  'grid w-full max-w-xl grid-cols-[minmax(0,1fr)_auto] items-end gap-4';

/** Field + two actions (poll / download). */
export const INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS =
  'grid w-full max-w-xl grid-cols-[minmax(0,1fr)_auto_auto] items-end gap-4';

/** Field + three actions (poll / cancel / download). */
export const INLINE_FILTER_ACTION_GRID_THREE_ACTIONS_CLASS =
  'grid w-full max-w-xl grid-cols-[minmax(0,1fr)_auto_auto_auto] items-end gap-4';

/** Two fields + one action on a single row. */
export const INLINE_FILTER_ACTION_GRID_TWO_FIELDS_CLASS =
  'grid w-full max-w-xl grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-end gap-4';

/** Stacked edit form on a page: cap the shell, not a field inside a full-width Card. */
export const NARROW_EDIT_FORM_CLASS = 'grid w-full max-w-xl gap-4';

/** Wider stacked edit form (textarea / JSON tools). */
export const NARROW_EDIT_FORM_WIDE_CLASS = 'grid w-full max-w-2xl gap-4';

/** Small field stack inside ui-filter-panel chrome. */
export const FILTER_PANEL_NARROW_CLASS = 'ui-filter-panel w-full max-w-xl';

/** Wider field stack inside ui-filter-panel chrome. */
export const FILTER_PANEL_WIDE_CLASS = 'ui-filter-panel w-full max-w-2xl';

/** Sparse read-only summary band (team overview, fraud preview). */
export const FILTER_PANEL_SUMMARY_CLASS =
  'ui-filter-panel w-full max-w-2xl gap-2 text-[13px] leading-[18px]';

/** Tighter vertical rhythm inside ui-filter-panel; keeps border, fill, and inset padding. */
export const FILTER_PANEL_FLAT_CLASS = 'gap-3';

/** Campaign editor main column beside paths aside. */
export const EDITOR_MAIN_COLUMN_CLASS = 'flex w-full max-w-3xl flex-col';

/** Inline controls (pagination, export) without stretching buttons to grid tracks. */
export const COMPACT_TOOLBAR_ROW_CLASS = 'flex flex-wrap items-center gap-3';

/** Bordered section panel (ops cards, overview blocks). */
export const SECTION_PANEL_CLASS = shellChrome.sectionPanelClass;

/** Raised surface inside page section stack. */
export const SECTION_SURFACE_RAISED_CLASS = shellChrome.surfaceRaisedClass;
