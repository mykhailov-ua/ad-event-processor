export const SIDEBAR_WIDTH_MIN = 200;
export const SIDEBAR_WIDTH_MAX = 360;
export const SIDEBAR_WIDTH_DEFAULT = 220;
export const SIDEBAR_COLLAPSED_WIDTH = 72;
export const SIDEBAR_MIN_MAIN_WIDTH = 640;
export const SIDEBAR_MAX_VIEWPORT_RATIO = 0.45;

export type SidebarWidthBounds = {
  min: number;
  max: number;
  default: number;
};

export function getSidebarWidthBounds(viewportWidth = window.innerWidth): SidebarWidthBounds {
  const vw = Math.max(320, viewportWidth);
  const maxByMain = vw - SIDEBAR_MIN_MAIN_WIDTH;
  const maxByRatio = Math.floor(vw * SIDEBAR_MAX_VIEWPORT_RATIO);
  const max = Math.max(SIDEBAR_WIDTH_MIN, Math.min(SIDEBAR_WIDTH_MAX, maxByMain, maxByRatio));
  const min = Math.min(SIDEBAR_WIDTH_MIN, max);
  return { min, max, default: SIDEBAR_WIDTH_DEFAULT };
}

export function clampSidebarWidth(width: number, viewportWidth = window.innerWidth): number {
  const { min, max } = getSidebarWidthBounds(viewportWidth);
  return Math.min(max, Math.max(min, Math.round(width)));
}
