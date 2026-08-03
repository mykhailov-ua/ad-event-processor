import { el, replaceChildren } from '../lib/dom.js';

/** @typedef {{ ctx: CanvasRenderingContext2D, width: number, height: number, dpr: number }} CanvasSurface */

export const CHART_HEIGHT_DEFAULT = 220;

let cachedRemPx = 16;
let remCachedAt = 0;

/** @type {Map<string, string>} */
const colorCache = new Map();

/** @type {Map<number, string>} */
const alphaCache = new Map();

/**
 * Invalidate CSS color / rem caches (theme toggle).
 */
export function invalidateChartThemeCache() {
  colorCache.clear();
  alphaCache.clear();
  remCachedAt = 0;
}

/**
 * Convert rem units to CSS pixels (cached ~250ms).
 *
 * @param {number} rem
 * @returns {number}
 */
export function remPx(rem) {
  const now = performance.now();
  if (now - remCachedAt > 250) {
    cachedRemPx = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
    remCachedAt = now;
  }
  return cachedRemPx * rem;
}

/**
 * Write standard chart padding into `out` (zero alloc).
 *
 * @param {{ top: number, right: number, bottom: number, left: number }} out
 * @param {{ top?: number, right?: number, bottom?: number, left?: number }} [overrides]
 */
export function chartPadInto(out, overrides = {}) {
  out.top = remPx(overrides.top ?? 0.375);
  out.right = remPx(overrides.right ?? 0.5);
  out.bottom = remPx(overrides.bottom ?? 1.75);
  out.left = remPx(overrides.left ?? 2.75);
}

const padReturn = { top: 0, right: 0, bottom: 0, left: 0 };

/**
 * @param {{ top?: number, right?: number, bottom?: number, left?: number }} [overrides]
 * @returns {{ top: number, right: number, bottom: number, left: number }}
 */
export function chartPad(overrides = {}) {
  chartPadInto(padReturn, overrides);
  return padReturn;
}

/** @type {string[]} */
export const CHART_SEGMENT_COLORS = [
  '--accent',
  '--success',
  '--warning',
  '--info',
  '--danger',
];

/**
 * Read a CSS custom property (cached).
 *
 * @param {string} name
 * @param {string} [fallback]
 * @returns {string}
 */
export function chartColor(name, fallback = '#888') {
  let v = colorCache.get(name);
  if (v === undefined) {
    v = getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
    colorCache.set(name, v);
  }
  return v;
}

/**
 * Resolve themed segment colors for pie / area charts (cold path).
 *
 * @param {number} count
 * @returns {string[]}
 */
export function segmentColors(count) {
  const out = new Array(count);
  for (let i = 0; i < count; i++) {
    out[i] = chartColor(CHART_SEGMENT_COLORS[i % CHART_SEGMENT_COLORS.length]);
  }
  return out;
}

/**
 * Configure a canvas for device-pixel-ratio aware drawing.
 * Read phase first (geometry), then write phase (canvas + styles).
 *
 * @param {HTMLElement} wrap
 * @param {HTMLCanvasElement} canvas
 * @param {number} [cssHeight]
 * @returns {CanvasSurface|null}
 */
export function setupCanvas(wrap, canvas, cssHeight) {
  const ctx = canvas.getContext('2d');
  if (!ctx) return null;

  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  const cssWidth = Math.max(wrap.clientWidth || 0, 12);
  const resolvedHeight = cssHeight ?? Math.max(wrap.clientHeight || 0, 80);
  const pxW = Math.round(cssWidth * dpr);
  const pxH = Math.round(resolvedHeight * dpr);

  canvas.style.width = `${cssWidth}px`;
  canvas.style.height = `${resolvedHeight}px`;

  if (canvas.width !== pxW || canvas.height !== pxH) {
    canvas.width = pxW;
    canvas.height = pxH;
  }

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  return { ctx, width: cssWidth, height: resolvedHeight, dpr };
}

/**
 * Mount a resize-aware canvas chart shell inside an aspect-ratio viewport.
 *
 * @param {HTMLElement} container
 * @param {string} ariaLabel
 * @param {number} [_cssHeight] legacy height hint (ignored; container drives size)
 * @returns {{ shell: HTMLDivElement, wrap: HTMLDivElement, canvas: HTMLCanvasElement, destroy: () => void, onResize: (fn: () => void) => void }}
 */
export function createChartShell(container, ariaLabel, _cssHeight = CHART_HEIGHT_DEFAULT) {
  let ro = null;
  let rafId = 0;
  /** @type {(() => void)|null} */
  let paintFn = null;

  const shell = el('div', { className: 'chart-shell' });
  const wrap = el('div', { className: 'chart-root chart-root--clip' });
  const canvas = el('canvas', {
    role: 'img',
    'aria-label': ariaLabel,
  });
  wrap.appendChild(canvas);
  shell.appendChild(wrap);
  replaceChildren(container, shell);

  function runPaint() {
    rafId = 0;
    if (paintFn) paintFn();
  }

  function schedulePaint() {
    if (!paintFn) return;
    if (rafId) return;
    rafId = requestAnimationFrame(runPaint);
  }

  ro = new ResizeObserver(schedulePaint);
  ro.observe(shell);

  return {
    shell,
    wrap,
    canvas,
    onResize(fn) {
      paintFn = fn;
      schedulePaint();
    },
    destroy() {
      if (rafId) cancelAnimationFrame(rafId);
      rafId = 0;
      ro.disconnect();
      ro = null;
      paintFn = null;
      container.replaceChildren();
    },
  };
}

export { alphaCache };
