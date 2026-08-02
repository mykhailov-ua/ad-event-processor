import { el, replaceChildren } from '../lib/dom.js';
import { seriesFromHourly } from '../helpers/chart_pool.js';

const CHART_HEIGHT = 280;
const PAD = { top: 16, right: 16, bottom: 28, left: 48 };

/**
 * Compute y-axis bounds for a series slice.
 *
 * @param {Float64Array} y
 * @param {number} length
 * @returns {{ yMin: number, yMax: number, invYRange: number }}
 */
function yScale(y, length) {
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
  return { yMin, yMax, invYRange: 1 / (yMax - yMin) };
}

/**
 * Draw a line series on a 2D canvas context using precomputed scales.
 *
 * @param {CanvasRenderingContext2D} ctx
 * @param {number} width
 * @param {number} height
 * @param {Float64Array} x
 * @param {Float64Array} y
 * @param {number} length
 * @param {{ xMin: number, invXRange: number, yMin: number, invYRange: number }} scale
 */
function drawLineChart(ctx, width, height, x, y, length, scale) {
  if (length < 1) return;

  const plotW = width - PAD.left - PAD.right;
  const plotH = height - PAD.top - PAD.bottom;

  ctx.strokeStyle = '#4a9eff';
  ctx.lineWidth = 2;
  ctx.beginPath();
  for (let i = 0; i < length; i++) {
    const px = PAD.left + (x[i] - scale.xMin) * scale.invXRange * plotW;
    const py = PAD.top + plotH - (y[i] - scale.yMin) * scale.invYRange * plotH;
    if (i === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  }
  ctx.stroke();

  ctx.strokeStyle = '#333';
  ctx.lineWidth = 1;
  ctx.strokeRect(PAD.left, PAD.top, plotW, plotH);
}

/**
 * Mount a canvas time-series chart for campaign hourly stats.
 *
 * @param {HTMLElement} container
 * @param {Array<{ hour: string, impressions?: number }>} hourly
 * @param {string} [label]
 * @returns {{ destroy: () => void }}
 */
export function mountChart(container, hourly, label = 'Impressions') {
  let ro = null;
  let rafId = 0;
  let lastWidth = 0;

  /**
   * Release canvas, observers, and pending animation frames.
   */
  function destroy() {
    if (rafId) cancelAnimationFrame(rafId);
    rafId = 0;
    ro?.disconnect();
    ro = null;
    container.replaceChildren();
  }

  if (!hourly?.length) {
    replaceChildren(container,
      el('div', { className: 'empty-state' },
        el('p', null, 'No data in period'),
        el('p', null, 'Hourly metrics appear after traffic.'),
      ),
    );
    return { destroy };
  }

  const { x, y, length } = seriesFromHourly(hourly, 'impressions');
  if (length === 0) {
    replaceChildren(container, el('p', null, 'No series points'));
    return { destroy };
  }

  const xMin = x[0];
  const xMax = x[length - 1] || xMin + 1;
  const { yMin, yMax, invYRange } = yScale(y, length);
  const scale = { xMin, invXRange: 1 / (xMax - xMin || 1), yMin, invYRange };

  const wrap = el('div', { className: 'chart-root' });
  const canvas = el('canvas', {
    width: 640,
    height: CHART_HEIGHT,
    role: 'img',
    'aria-label': `${label} chart`,
  });
  wrap.appendChild(canvas);
  replaceChildren(container, wrap);

  const ctx = canvas.getContext('2d');

  /**
   * Resize canvas when width changes and redraw the series.
   */
  function paint() {
    if (!ctx) return;
    const width = Math.max(wrap.clientWidth || 640, 320);
    if (width !== lastWidth) {
      lastWidth = width;
      canvas.width = width;
      canvas.height = CHART_HEIGHT;
    }
    ctx.clearRect(0, 0, width, CHART_HEIGHT);
    drawLineChart(ctx, width, CHART_HEIGHT, x, y, length, scale);
  }

  /**
   * Coalesce resize events into one animation frame paint.
   */
  function schedulePaint() {
    if (rafId) return;
    rafId = requestAnimationFrame(() => {
      rafId = 0;
      paint();
    });
  }

  paint();
  ro = new ResizeObserver(() => schedulePaint());
  ro.observe(wrap);

  return { destroy };
}
