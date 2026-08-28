import { el, replaceChildren } from '../lib/dom.js';
import type { PadBox } from './chart_math.js';

export type CanvasSurface = {
  ctx: CanvasRenderingContext2D;
  width: number;
  height: number;
  dpr: number;
};

export type ChartPadOverrides = {
  top?: number;
  right?: number;
  bottom?: number;
  left?: number;
};

export type ChartShell = {
  shell: HTMLDivElement;
  wrap: HTMLDivElement;
  canvas: HTMLCanvasElement;
  destroy: () => void;
  onResize: (fn: () => void) => void;
};

export const CHART_HEIGHT_DEFAULT = 220;

let cachedRemPx = 16;
let remCachedAt = 0;

const colorCache = new Map<string, string>();
const alphaCache = new Map<number, string>();

export function invalidateChartThemeCache(): void {
  colorCache.clear();
  alphaCache.clear();
  remCachedAt = 0;
}

export function remPx(rem: number): number {
  const now = performance.now();
  if (now - remCachedAt > 250) {
    cachedRemPx = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
    remCachedAt = now;
  }
  return cachedRemPx * rem;
}

export function chartPadInto(out: PadBox, overrides: ChartPadOverrides = {}): void {
  out.top = remPx(overrides.top ?? 0.375);
  out.right = remPx(overrides.right ?? 2.25);
  out.bottom = remPx(overrides.bottom ?? 1.75);
  out.left = remPx(overrides.left ?? 2.25);
}

const padReturn: PadBox = { top: 0, right: 0, bottom: 0, left: 0 };

export function chartPad(overrides: ChartPadOverrides = {}): PadBox {
  chartPadInto(padReturn, overrides);
  return padReturn;
}

export const CHART_SEGMENT_COLORS: string[] = [
  '--accent',
  '--success',
  '--warning',
  '--info',
  '--danger',
];

export function chartColor(name: string, fallback = '#888'): string {
  let v = colorCache.get(name);
  if (v === undefined) {
    v = getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
    colorCache.set(name, v);
  }
  return v;
}

export function segmentColors(count: number): string[] {
  const out = new Array<string>(count);
  for (let i = 0; i < count; i++) {
    out[i] = chartColor(CHART_SEGMENT_COLORS[i % CHART_SEGMENT_COLORS.length]);
  }
  return out;
}

export function setupCanvas(
  wrap: HTMLElement,
  canvas: HTMLCanvasElement,
  cssHeight?: number
): CanvasSurface | null {
  const ctx = canvas.getContext('2d', { alpha: true });
  if (!ctx) return null;

  const dpr = Math.max(window.devicePixelRatio || 1, 2);
  const cssWidth = Math.max(wrap.clientWidth || 0, 12);
  const resolvedHeight = cssHeight ?? Math.max(wrap.clientHeight || 0, 80);
  const pxW = Math.round(cssWidth * dpr);
  const pxH = Math.round(resolvedHeight * dpr);

  if (canvas.style.width !== `${cssWidth}px`) {
    canvas.style.width = `${cssWidth}px`;
  }
  if (canvas.style.height !== `${resolvedHeight}px`) {
    canvas.style.height = `${resolvedHeight}px`;
  }

  if (canvas.width !== pxW || canvas.height !== pxH) {
    canvas.width = pxW;
    canvas.height = pxH;
  }

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = 'high';
  return { ctx, width: cssWidth, height: resolvedHeight, dpr };
}

export function createChartShell(
  container: HTMLElement,
  ariaLabel: string,
  _cssHeight = CHART_HEIGHT_DEFAULT
): ChartShell {
  let ro: ResizeObserver | null = null;
  let rafId = 0;
  let paintFn: (() => void) | null = null;

  const shell = el('div', { className: 'chart-shell' }) as HTMLDivElement;
  const wrap = el('div', { className: 'chart-root chart-root--clip' }) as HTMLDivElement;
  const canvas = el('canvas', {
    role: 'img',
    'aria-label': ariaLabel,
  }) as HTMLCanvasElement;
  wrap.appendChild(canvas);
  shell.appendChild(wrap);
  replaceChildren(container, shell);

  function runPaint(): void {
    rafId = 0;
    if (paintFn) paintFn();
  }

  function schedulePaint(): void {
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
    onResize(fn: () => void) {
      paintFn = fn;
      schedulePaint();
    },
    destroy() {
      if (rafId) cancelAnimationFrame(rafId);
      rafId = 0;
      if (ro) ro.disconnect();
      ro = null;
      paintFn = null;
      container.replaceChildren();
    },
  };
}

export { alphaCache };
