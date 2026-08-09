import { el, replaceChildren } from '../lib/dom.js';
import { formatChartAxisTime } from '../helpers/chart_format.js';
import { seriesFromHourly } from '../helpers/chart_pool.js';
import { SERIES_CAP, padScratch } from './chart_math.js';
import {
  chartColor,
  chartPadInto,
  createChartShell,
  remPx,
  setupCanvas,
} from './canvas_util.js';

const FONT_AXIS = 'var(--text-xs) var(--font-family), system-ui, sans-serif';

/**
 * Compute y-axis bounds for a series slice.
 *
 * @param {Float64Array} y
 * @param {number} length
 * @param {{ yMin: number, yMax: number, invYRange: number }} out
 */
function yScaleInto(y, length, out) {
  let yMin = y[0];
  let yMax = y[0];
  for (let i = 1; i < length; i++) {
    const v = y[i];
    if (v < yMin) yMin = v;
    if (v > yMax) yMax = v;
  }
  if (yMax === yMin) {
    yMax += 1;
    yMin -= 1;
  }
  out.yMin = yMin;
  out.yMax = yMax;
  out.invYRange = 1 / (yMax - yMin);
}

/**
 * Draw a straight line series on a 2D canvas context using precomputed scales.
 *
 * @param {CanvasRenderingContext2D} ctx
 * @param {number} width
 * @param {number} height
 * @param {Float64Array} x
 * @param {Float64Array} y
 * @param {number} length
 * @param {{ xMin: number, invXRange: number, yMin: number, invYRange: number }} scale
 * @param {Float64Array} ptsX
 * @param {Float64Array} ptsY
 * @param {string} accent
 * @param {string} border
 * @param {string} muted
 */
function drawLineChart(ctx, width, height, x, y, length, scale, ptsX, ptsY, accent, border, muted) {
  if (length < 1) return;

  chartPadInto(padScratch);
  const padTop = padScratch.top;
  const padRight = padScratch.right;
  const padBottom = padScratch.bottom;
  const padLeft = padScratch.left;
  const plotW = Math.max(width - padLeft - padRight, 10);
  const plotH = Math.max(height - padTop - padBottom, 10);
  const baseY = padTop + plotH;

  for (let i = 0; i < length; i++) {
    ptsX[i] = padLeft + (x[i] - scale.xMin) * scale.invXRange * plotW;
    ptsY[i] = padTop + plotH - (y[i] - scale.yMin) * scale.invYRange * plotH;
  }

  const grad = ctx.createLinearGradient(0, padTop, 0, baseY);
  grad.addColorStop(0, chartColor('--accent-subtle', 'rgba(77,143,232,0.22)'));
  grad.addColorStop(1, 'rgba(0,0,0,0)');

  ctx.beginPath();
  ctx.moveTo(ptsX[0], baseY);
  for (let i = 0; i < length; i++) {
    ctx.lineTo(ptsX[i], ptsY[i]);
  }
  ctx.lineTo(ptsX[length - 1], baseY);
  ctx.closePath();
  ctx.fillStyle = grad;
  ctx.fill();

  ctx.strokeStyle = accent;
  ctx.lineWidth = 2;
  ctx.lineJoin = 'round';
  ctx.lineCap = 'round';
  ctx.beginPath();
  ctx.moveTo(ptsX[0], ptsY[0]);
  for (let i = 1; i < length; i++) {
    ctx.lineTo(ptsX[i], ptsY[i]);
  }
  ctx.stroke();

  ctx.strokeStyle = border;
  ctx.lineWidth = 1;
  ctx.strokeRect(padLeft, padTop, plotW, plotH);

  const tickCount = length < 5 ? length : 5;
  ctx.fillStyle = muted;
  ctx.font = FONT_AXIS;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'top';
  const labelY = height - padBottom + remPx(0.125);
  const inv = tickCount === 1 ? 0 : 1 / (tickCount - 1);
  for (let i = 0; i < tickCount; i++) {
    const idx = tickCount === 1 ? 0 : Math.round(i * inv * (length - 1));
    ctx.fillText(formatChartAxisTime(x[idx] * 1000, 24 * 60 * 60 * 1000), ptsX[idx], labelY);
  }
}

/**
 * Mount a canvas time-series chart for campaign hourly stats.
 *
 * @param {HTMLElement} container
 * @param {Array<{ hour: string, impressions?: number, clicks?: number, conversions?: number }>} hourly
 * @param {{ field?: string, label?: string }} [options]
 * @returns {{ destroy: () => void }}
 */
export function mountChart(container, hourly, options = {}) {
  const field = options.field ?? 'impressions';
  const label = options.label ?? field.replace(/_/g, ' ');

  if (!hourly?.length) {
    replaceChildren(container,
      el('div', { className: 'empty-state' },
        el('p', null, 'No data in period'),
        el('p', null, 'Hourly metrics appear after traffic.'),
      ),
    );
    return { destroy: () => container.replaceChildren() };
  }

  const { x, y, length } = seriesFromHourly(hourly, field);
  if (length === 0) {
    replaceChildren(container, el('p', null, 'No series points'));
    return { destroy: () => container.replaceChildren() };
  }

  const xMin = x[0];
  const xMax = x[length - 1] || xMin + 1;
  const yScaleOut = { yMin: 0, yMax: 1, invYRange: 1 };
  yScaleInto(y, length, yScaleOut);
  const scale = {
    xMin,
    invXRange: 1 / (xMax - xMin || 1),
    yMin: yScaleOut.yMin,
    invYRange: yScaleOut.invYRange,
  };

  const ptsX = new Float64Array(SERIES_CAP);
  const ptsY = new Float64Array(SERIES_CAP);
  const accent = chartColor('--accent');
  const border = chartColor('--border-color');
  const muted = chartColor('--text-muted');

  const shell = createChartShell(container, `${label} chart`);

  shell.onResize(() => {
    const cssHeight = Math.max(shell.wrap.clientHeight, remPx(5));
    const surface = setupCanvas(shell.wrap, shell.canvas, cssHeight);
    if (!surface) return;
    const { ctx, width, height } = surface;
    ctx.clearRect(0, 0, width, height);
    drawLineChart(ctx, width, height, x, y, length, scale, ptsX, ptsY, accent, border, muted);
  });

  return { destroy: () => shell.destroy() };
}

/**
 * Mount a line chart from forecast spend-curve points.
 *
 * @param {HTMLElement} container
 * @param {Array<{ hour: string, spend_micro?: number, impressions?: number }>} curve
 * @param {'impressions'|'spend_micro'} [field]
 * @returns {{ destroy: () => void }}
 */
export function mountSpendCurveChart(container, curve, field = 'impressions') {
  const hourly = [];
  const src = curve ?? [];
  for (let i = 0; i < src.length; i++) {
    const point = src[i];
    hourly.push({
      hour: point.hour,
      [field]: Number(point[field]) || 0,
    });
  }
  const chartLabel = field === 'spend_micro' ? 'Projected spend' : 'Projected impressions';
  return mountChart(container, hourly, { field, label: chartLabel });
}
