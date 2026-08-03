import { el, replaceChildren } from '../lib/dom.js';
import {
  SERIES_CAP,
  fillSmoothAreaSoA,
  padScratch,
  strokeSeriesLineSoA,
} from './chart_math.js';
import {
  chartColor,
  chartPadInto,
  createChartShell,
  remPx,
  segmentColors,
  setupCanvas,
} from './canvas_util.js';

const FONT_AXIS = 'var(--text-xs) var(--font-family), system-ui, sans-serif';

/**
 * Mount a HiDPI smooth area chart for categorical metrics.
 *
 * @param {HTMLElement} container
 * @param {Array<{ label: string, value: number }>} items
 * @param {string} [ariaLabel]
 * @returns {{ destroy: () => void }}
 */
export function mountAreaChart(container, items, ariaLabel = 'Area chart') {
  const rows = [];
  const src = items ?? [];
  for (let i = 0; i < src.length; i++) {
    const item = src[i];
    if (item && item.label != null) rows.push(item);
  }
  if (rows.length === 0) {
    replaceChildren(container, el('p', { className: 'text-muted text-sm' }, 'No data to chart.'));
    return { destroy: () => container.replaceChildren() };
  }

  const n = rows.length;
  const ptsX = new Float64Array(SERIES_CAP);
  const ptsY = new Float64Array(SERIES_CAP);
  const labels = new Array(n);
  const values = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    labels[i] = rows[i].label;
    values[i] = Number(rows[i].value) || 0;
  }

  const shell = createChartShell(container, ariaLabel);
  const accent = segmentColors(1)[0];
  const borderCached = chartColor('--border-color');
  const mutedCached = chartColor('--text-muted');
  const surfaceCached = chartColor('--bg-surface');

  function paint() {
    chartPadInto(padScratch, { left: 1.25, right: 1.25, top: 1.25, bottom: 2.25 });
    const padTop = padScratch.top;
    const padRight = padScratch.right;
    const padBottom = padScratch.bottom;
    const padLeft = padScratch.left;

    const cssHeight = Math.max(shell.wrap.clientHeight, remPx(5));
    const surface = setupCanvas(shell.wrap, shell.canvas, cssHeight);
    if (!surface) return;

    const ctx = surface.ctx;
    const width = surface.width;
    const height = surface.height;

    let maxVal = 0;
    for (let i = 0; i < n; i++) {
      const v = values[i];
      if (v > maxVal) maxVal = v;
    }
    if (maxVal <= 0) maxVal = 1;

    const plotW = width - padLeft - padRight;
    const plotH = height - padTop - padBottom;
    const baseY = padTop + plotH;
    const cellW = n === 1 ? plotW : plotW / n;

    for (let i = 0; i < n; i++) {
      const val = values[i];
      ptsX[i] = padLeft + (n === 1 ? plotW * 0.5 : (i + 0.5) * cellW);
      ptsY[i] = padTop + plotH - (val / maxVal) * plotH;
    }

    ctx.clearRect(0, 0, width, height);
    ctx.strokeStyle = borderCached;
    ctx.lineWidth = 1;
    ctx.strokeRect(padLeft, padTop, plotW, plotH);

    ctx.save();
    ctx.beginPath();
    ctx.rect(padLeft, padTop, plotW, plotH);
    ctx.clip();

    const grad = ctx.createLinearGradient(0, padTop, 0, baseY);
    grad.addColorStop(0, chartColor('--accent-subtle', 'rgba(77,143,232,0.25)'));
    grad.addColorStop(1, 'rgba(0,0,0,0)');
    ctx.fillStyle = grad;
    fillSmoothAreaSoA(ctx, ptsX, ptsY, n, baseY);

    ctx.strokeStyle = accent;
    ctx.lineWidth = 2;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';
    strokeSeriesLineSoA(ctx, ptsX, ptsY, n);

    const dotR = remPx(0.25);
    for (let i = 0; i < n; i++) {
      ctx.beginPath();
      ctx.arc(ptsX[i], ptsY[i], dotR, 0, Math.PI * 2);
      ctx.fillStyle = accent;
      ctx.fill();
      ctx.strokeStyle = surfaceCached;
      ctx.lineWidth = 2;
      ctx.stroke();
    }
    ctx.restore();

    ctx.fillStyle = mutedCached;
    ctx.font = FONT_AXIS;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    const labelY = height - padBottom + remPx(0.125);
    for (let i = 0; i < n; i++) {
      const raw = labels[i];
      const label = raw.length > 12 ? `${raw.slice(0, 11)}…` : raw;
      ctx.fillText(label, ptsX[i], labelY);
    }
  }

  shell.onResize(paint);

  return { destroy: () => shell.destroy() };
}
