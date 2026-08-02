import { describe, expect, it } from 'vitest';
import {
  clampSidebarWidth,
  getSidebarWidthBounds,
  SIDEBAR_WIDTH_MIN,
} from './sidebar_layout.js';

describe('sidebar_layout', () => {
  it('clamps width between min and viewport-derived max', () => {
    const bounds = getSidebarWidthBounds(1600);
    expect(bounds.min).toBe(SIDEBAR_WIDTH_MIN);
    expect(bounds.max).toBeLessThanOrEqual(400);
    expect(bounds.max).toBeGreaterThan(bounds.min);
    expect(clampSidebarWidth(50, 1600)).toBe(bounds.min);
    expect(clampSidebarWidth(999, 1600)).toBe(bounds.max);
  });

  it('limits sidebar on narrow viewports to preserve main area', () => {
    const bounds = getSidebarWidthBounds(1000);
    expect(bounds.max).toBeLessThanOrEqual(1000 - 640);
  });
});
