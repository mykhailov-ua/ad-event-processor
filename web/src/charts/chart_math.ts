export type YDomain = {
  min: number;
  max: number;
  flat: boolean;
};

export type PadBox = {
  top: number;
  right: number;
  bottom: number;
  left: number;
};

export const SERIES_CAP = 512;
export const Y_TICK_CAP = 8;

/** Reused padding box — single paint at a time per main thread. */
export const padScratch: PadBox = { top: 0, right: 0, bottom: 0, left: 0 };

/** Reused Y-domain result. */
export const yDomainScratch: YDomain = { min: 0, max: 1, flat: false };

/**
 * Round a positive value up to a nice tick ceiling.
 */
export function niceCeil(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 1;
  const exp = Math.floor(Math.log10(value));
  const base = 10 ** exp;
  const frac = value / base;
  let nice: number;
  if (frac <= 1) nice = 1;
  else if (frac <= 2) nice = 2;
  else if (frac <= 5) nice = 5;
  else nice = 10;
  return nice * base;
}

/**
 * Fill tick buffer; returns tick count (no allocations).
 */
export function fillNiceTicks(min: number, max: number, out: Float64Array, count = 4): number {
  const span = Math.max(max - min, 1e-9);
  const roughStep = span / Math.max(count - 1, 1);
  const step = niceCeil(roughStep);
  const start = Math.floor(min / step) * step;
  let n = 0;
  const limit = out.length;
  for (let t = start; t <= max + step * 0.001 && n < limit; t += step) {
    if (t >= min - step * 0.001) {
      out[n] = +t.toFixed(10);
      n++;
    }
  }
  if (n === 0) {
    out[0] = min;
    if (limit > 1) {
      out[1] = max;
      return 2;
    }
    return 1;
  }
  return n;
}

/**
 * Scan SoA series for Y domain; writes into `out`.
 */
export function computeYDomainSoA(
  values: Float64Array,
  len: number,
  optsMin: number,
  optsMax: number,
  out: YDomain,
): void {
  let dataMin = Infinity;
  let dataMax = -Infinity;
  for (let i = 0; i < len; i++) {
    const v = values[i];
    if (v < dataMin) dataMin = v;
    if (v > dataMax) dataMax = v;
  }
  if (!Number.isFinite(dataMin)) {
    out.min = 0;
    out.max = 1;
    out.flat = true;
    return;
  }

  const flat = len < 2
    || (dataMax - dataMin) < Math.max(Math.abs(dataMax) * 1e-6, 1e-9);

  if (Number.isFinite(optsMin) && Number.isFinite(optsMax)) {
    out.min = optsMin;
    out.max = optsMax;
    out.flat = flat;
    return;
  }

  if (flat) {
    const v = dataMax;
    if (v === 0) {
      out.min = 0;
      out.max = 1;
      out.flat = true;
      return;
    }
    const pad = Math.max(Math.abs(v) * 0.2, niceCeil(Math.abs(v) * 0.1) * 0.5, 0.5);
    const min = v >= 0 ? Math.max(0, v - pad) : v - pad;
    const max = v + pad;
    out.min = min;
    out.max = max > min ? max : min + 1;
    out.flat = true;
    return;
  }

  const beginAtZero = dataMin >= 0;
  const min = beginAtZero ? 0 : dataMin;
  let max = niceCeil(Math.max(dataMax * 1.08, dataMax + 1e-9, 1));
  if (max <= min) max = min + 1;
  out.min = min;
  out.max = max;
  out.flat = false;
}

/**
 * Project time series into plot coordinates (SoA → SoA).
 */
export function projectSeriesSoA(
  tsIn: Float64Array,
  valIn: Float64Array,
  len: number,
  xMin: number,
  xSpan: number,
  plotW: number,
  yMin: number,
  ySpan: number,
  padLeft: number,
  padTop: number,
  plotH: number,
  outX: Float64Array,
  outY: Float64Array,
): number {
  const invX = plotW / Math.max(xSpan, 1);
  const invY = plotH / Math.max(ySpan, 1e-9);
  const baseY = padTop + plotH;
  let n = 0;
  const cap = outX.length;
  for (let i = 0; i < len && n < cap; i++) {
    outX[n] = padLeft + (tsIn[i] - xMin) * invX;
    outY[n] = baseY - (valIn[i] - yMin) * invY;
    n++;
  }
  return n;
}

/**
 * Draw straight point-to-point line series (no Bezier curves).
 */
export function strokeSeriesLineSoA(
  ctx: CanvasRenderingContext2D,
  xs: Float64Array,
  ys: Float64Array,
  len: number,
  dashed = false,
): void {
  if (len < 2) return;
  if (dashed) ctx.setLineDash([5, 4]);
  ctx.beginPath();
  ctx.moveTo(xs[0], ys[0]);
  for (let i = 1; i < len; i++) {
    ctx.lineTo(xs[i], ys[i]);
  }
  ctx.stroke();
  if (dashed) ctx.setLineDash([]);
}

/**
 * Fill area under straight line series (no Bezier curves).
 */
export function fillSmoothAreaSoA(
  ctx: CanvasRenderingContext2D,
  xs: Float64Array,
  ys: Float64Array,
  len: number,
  baseY: number,
): void {
  if (len < 2) return;
  ctx.beginPath();
  ctx.moveTo(xs[0], baseY);
  for (let i = 0; i < len; i++) {
    ctx.lineTo(xs[i], ys[i]);
  }
  ctx.lineTo(xs[len - 1], baseY);
  ctx.closePath();
  ctx.fill();
}

/**
 * Parse #RRGGBB to rgba string; caches in module table keyed by packed key.
 */
export function withAlphaCached(cssColor: string, alpha: number, cache: Map<number, string>): string {
  if (!cssColor.startsWith('#') || cssColor.length < 7) return cssColor;
  const r = parseInt(cssColor.slice(1, 3), 16);
  const g = parseInt(cssColor.slice(3, 5), 16);
  const b = parseInt(cssColor.slice(5, 7), 16);
  const key = (r << 24) | (g << 16) | (b << 8) | ((alpha * 100) | 0);
  let hit = cache.get(key);
  if (hit === undefined) {
    hit = `rgba(${r},${g},${b},${alpha})`;
    cache.set(key, hit);
  }
  return hit;
}
