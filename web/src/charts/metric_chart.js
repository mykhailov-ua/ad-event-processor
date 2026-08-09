import { el, replaceChildren } from '../lib/dom.js';
import { formatChartTick, formatChartAxisTime } from '../helpers/chart_format.js';
import { copyMetricPoints, rangeMsFromHours } from '../helpers/ops_metric_series.js';
import {
  SERIES_CAP,
  Y_TICK_CAP,
  padScratch,
  yDomainScratch,
  fillNiceTicks,
  computeYDomainSoA,
  projectSeriesSoA,
  strokeSeriesLineSoA,
  fillSmoothAreaSoA,
  withAlphaCached,
} from './chart_math.js';
import {
  alphaCache,
  chartColor,
  chartPadInto,
  createChartShell,
  remPx,
  setupCanvas,
} from './canvas_util.js';

export const CHART_HEIGHT_METRIC = 200;

const X_TICK_MAX = 5;
const FONT_AXIS = 'var(--text-xs) var(--font-family), system-ui, sans-serif';

/**
 * Mount a Grafana-style time-series area chart for a single metric.
 * Pixel-snapped grid lines (0.5px offset) eliminate staircase/aliasing artifacts.
 *
 * @param {HTMLElement} container
 * @param {{
 *   title: string,
 *   points?: Array<{ ts: number, value: number }>,
 *   value?: number,
 *   min?: number,
 *   max?: number,
 *   rangeHours?: number,
 *   color?: string,
 *   formatValue?: (value: number) => string,
 * }} opts
 * @returns {{ destroy: () => void, update: (next: object) => void }}
 */
export function mountMetricChart(container, opts) {
  const title = opts.title ?? 'Metric';
  let colorToken = opts.color ?? '--accent';
  let yMinOpt = Number.isFinite(opts.min) ? opts.min : NaN;
  let yMaxOpt = Number.isFinite(opts.max) ? opts.max : NaN;
  let rangeMs = rangeMsFromHours(opts.rangeHours ?? 24);
  let fallbackValue = Number(opts.value) || 0;

  const seriesTs = new Float64Array(SERIES_CAP);
  const seriesVal = new Float64Array(SERIES_CAP);
  const ptsX = new Float64Array(SERIES_CAP);
  const ptsY = new Float64Array(SERIES_CAP);
  const yTicks = new Float64Array(Y_TICK_CAP);
  let seriesLen = 0;

  let accentCached = chartColor(colorToken);
  let mutedCached = chartColor('--text-muted');
  let gridMajorCached = withAlphaCached(chartColor('--border-color'), 0.55, alphaCache);
  let gridMinorCached = withAlphaCached(chartColor('--border-color'), 0.28, alphaCache);
  let surfaceCached = chartColor('--bg-surface');

  /** @type {Array<{ ts: number, value: number }>|null} */
  let pointsRef = opts.points ?? null;

  seriesLen = copyMetricPoints(pointsRef, fallbackValue, rangeMs, seriesTs, seriesVal);

  const shell = createChartShell(container, title, CHART_HEIGHT_METRIC);

  function refreshThemeColors() {
    accentCached = chartColor(colorToken);
    mutedCached = chartColor('--text-muted');
    const border = chartColor('--border-color');
    gridMajorCached = withAlphaCached(border, 0.55, alphaCache);
    gridMinorCached = withAlphaCached(border, 0.28, alphaCache);
    surfaceCached = chartColor('--bg-surface');
  }

  function reloadSeries(now) {
    seriesLen = copyMetricPoints(pointsRef, fallbackValue, rangeMs, seriesTs, seriesVal, now);
  }

  function paint() {
    chartPadInto(padScratch);
    const padTop = padScratch.top;
    const padRight = padScratch.right;
    const padBottom = padScratch.bottom;
    const padLeft = padScratch.left;

    const cssHeight = Math.max(shell.wrap.clientHeight, 80);
    const surface = setupCanvas(shell.wrap, shell.canvas, cssHeight);
    if (!surface) return;

    const ctx = surface.ctx;
    const width = surface.width;
    const height = surface.height;
    const plotW = Math.max(width - padLeft - padRight, 10);
    const plotH = Math.max(height - padTop - padBottom, 10);
    const baseY = padTop + plotH;

    const now = Date.now();
    const xMin = now - rangeMs;
    const xSpan = rangeMs;

    reloadSeries(now);

    computeYDomainSoA(seriesVal, seriesLen, yMinOpt, yMaxOpt, yDomainScratch);
    const yMin = yDomainScratch.min;
    const yMax = yDomainScratch.max;
    const flat = yDomainScratch.flat;
    const ySpan = yMax - yMin;

    const ptLen = projectSeriesSoA(
      seriesTs, seriesVal, seriesLen,
      xMin, xSpan, plotW,
      yMin, ySpan, padLeft, padTop, plotH,
      ptsX, ptsY,
    );

    ctx.clearRect(0, 0, width, height);

    // Y Grid & Tick Labels with 0.5px Pixel-Snapping (Zero Staircase/Aliasing)
    const yTickCount = fillNiceTicks(yMin, yMax, yTicks, 4);
    ctx.font = FONT_AXIS;
    ctx.textBaseline = 'middle';
    const labelPad = remPx(0.375);
    for (let i = 0; i < yTickCount; i++) {
      const t = yTicks[i];
      const rawY = padTop + plotH - ((t - yMin) / ySpan) * plotH;
      const snapY = Math.floor(rawY) + 0.5;
      ctx.strokeStyle = i === 0 ? gridMajorCached : gridMinorCached;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(padLeft, snapY);
      ctx.lineTo(padLeft + plotW, snapY);
      ctx.stroke();

      ctx.textAlign = 'right';
      ctx.fillStyle = mutedCached;
      ctx.fillText(formatChartTick(t), padLeft - labelPad, rawY);
    }

    // X Grid & Time Tick Labels with 0.5px Pixel-Snapping
    const xLabelY = height - padBottom + remPx(0.25);
    ctx.textBaseline = 'top';
    const xInv = X_TICK_MAX === 1 ? 0 : 1 / (X_TICK_MAX - 1);
    for (let i = 0; i < X_TICK_MAX; i++) {
      const ratio = X_TICK_MAX === 1 ? 0 : i * xInv;
      const ts = xMin + ratio * xSpan;
      const rawX = padLeft + ratio * plotW;
      const snapX = Math.floor(rawX) + 0.5;
      ctx.strokeStyle = gridMinorCached;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(snapX, padTop);
      ctx.lineTo(snapX, baseY);
      ctx.stroke();

      ctx.fillStyle = mutedCached;
      if (i === 0) {
        ctx.textAlign = 'left';
      } else if (i === X_TICK_MAX - 1) {
        ctx.textAlign = 'right';
      } else {
        ctx.textAlign = 'center';
      }
      ctx.fillText(formatChartAxisTime(ts, rangeMs), rawX, xLabelY);
    }

    // Pixel-Snapped Outer Plot Frame
    ctx.strokeStyle = gridMajorCached;
    ctx.lineWidth = 1;
    const snapLeft = Math.floor(padLeft) + 0.5;
    const snapTop = Math.floor(padTop) + 0.5;
    ctx.strokeRect(snapLeft, snapTop, Math.floor(plotW), Math.floor(plotH));

    // Render Data Series Line & Area Fill
    ctx.save();
    ctx.beginPath();
    ctx.rect(padLeft, padTop, plotW, plotH);
    ctx.clip();

    if (!flat) {
      const grad = ctx.createLinearGradient(0, padTop, 0, baseY);
      grad.addColorStop(0, withAlphaCached(accentCached, 0.32, alphaCache));
      grad.addColorStop(0.55, withAlphaCached(accentCached, 0.1, alphaCache));
      grad.addColorStop(1, withAlphaCached(accentCached, 0, alphaCache));
      ctx.fillStyle = grad;
      fillSmoothAreaSoA(ctx, ptsX, ptsY, ptLen, baseY);
    }

    ctx.strokeStyle = accentCached;
    ctx.lineWidth = flat ? 1.5 : 2;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';
    strokeSeriesLineSoA(ctx, ptsX, ptsY, ptLen, flat);

    if (ptLen > 0 && !flat) {
      const lastX = ptsX[ptLen - 1];
      const lastY = ptsY[ptLen - 1];
      ctx.beginPath();
      ctx.arc(lastX, lastY, 3, 0, Math.PI * 2);
      ctx.fillStyle = accentCached;
      ctx.fill();
      ctx.strokeStyle = surfaceCached;
      ctx.lineWidth = 1.5;
      ctx.stroke();
    }
    ctx.restore();
  }

  shell.onResize(paint);

  return {
    update(next) {
      if (next.color) {
        colorToken = next.color;
        refreshThemeColors();
      }
      if (Number.isFinite(next.min)) yMinOpt = next.min;
      if (Number.isFinite(next.max)) yMaxOpt = next.max;
      if (next.rangeHours != null) rangeMs = rangeMsFromHours(next.rangeHours);
      if (next.value != null) fallbackValue = Number(next.value) || 0;
      if (next.points) pointsRef = next.points;
      paint();
    },
    destroy: () => shell.destroy(),
  };
}

/**
 * @param {HTMLElement} container
 * @returns {{ destroy: () => void }}
 */
export function mountMetricChartEmpty(container) {
  replaceChildren(container, el('p', { className: 'text-muted text-sm' }, 'No data to chart.'));
  return { destroy: () => container.replaceChildren() };
}
