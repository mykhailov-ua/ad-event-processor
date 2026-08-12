/** Minimum sidebar width — keeps nav labels readable. */
export const SIDEBAR_WIDTH_MIN = 232;

/** Hard cap — sidebar never wider than this. */
export const SIDEBAR_WIDTH_MAX = 400;

export const SIDEBAR_WIDTH_DEFAULT = 240;

export const SIDEBAR_COLLAPSED_WIDTH = 72;

/** Minimum main dashboard width (px). */
export const SIDEBAR_MIN_MAIN_WIDTH = 640;

/** Sidebar cannot exceed this share of the viewport. */
export const SIDEBAR_MAX_VIEWPORT_RATIO = 0.45;

export type SidebarWidthBounds = {
  min: number;
  max: number;
  default: number;
};

/**
 * Compute min, max, and default sidebar widths for a viewport.
 */
export function getSidebarWidthBounds(viewportWidth = window.innerWidth): SidebarWidthBounds {
  const vw = Math.max(320, viewportWidth);
  const maxByMain = vw - SIDEBAR_MIN_MAIN_WIDTH;
  const maxByRatio = Math.floor(vw * SIDEBAR_MAX_VIEWPORT_RATIO);
  const max = Math.max(
    SIDEBAR_WIDTH_MIN,
    Math.min(SIDEBAR_WIDTH_MAX, maxByMain, maxByRatio),
  );
  const min = Math.min(SIDEBAR_WIDTH_MIN, max);
  return { min, max, default: SIDEBAR_WIDTH_DEFAULT };
}

/**
 * Clamp a sidebar width to allowed bounds for the viewport.
 */
export function clampSidebarWidth(width: number, viewportWidth = window.innerWidth): number {
  const { min, max } = getSidebarWidthBounds(viewportWidth);
  return Math.min(max, Math.max(min, Math.round(width)));
}
