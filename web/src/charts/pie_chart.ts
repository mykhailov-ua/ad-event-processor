import { el, replaceChildren } from '../lib/dom.js';
import {
  CHART_HEIGHT_DEFAULT,
  chartColor,
  createChartShell,
  segmentColors,
  setupCanvas,
} from './canvas_util.js';
import type { ChartCategoryItem, ChartHandle } from './area_chart.js';

/**
 * Draw a single donut slice between start and end angles.
 */
function drawDonutSlice(
  ctx: CanvasRenderingContext2D,
  cx: number,
  cy: number,
  outerR: number,
  innerR: number,
  start: number,
  end: number,
  color: string,
): void {
  ctx.beginPath();
  ctx.arc(cx, cy, outerR, start, end);
  ctx.arc(cx, cy, innerR, end, start, true);
  ctx.closePath();
  ctx.fillStyle = color;
  ctx.fill();
}

/**
 * Mount a HiDPI donut chart with an HTML legend.
 */
export function mountPieChart(
  container: HTMLElement,
  items: ChartCategoryItem[] | null | undefined,
  ariaLabel = 'Donut chart',
): ChartHandle {
  const rows = (items ?? []).filter((item): item is ChartCategoryItem => item != null && item.label != null);
  const total = rows.reduce((sum, item) => sum + (Number(item.value) || 0), 0);

  if (rows.length === 0 || total <= 0) {
    replaceChildren(container, el('p', { className: 'text-muted text-sm' }, 'No data to chart.'));
    return { destroy: () => container.replaceChildren() };
  }

  const shell = createChartShell(container, ariaLabel, CHART_HEIGHT_DEFAULT + 8);
  const legend = el('div', { className: 'chart-legend' });
  shell.wrap.appendChild(legend);

  const colors = segmentColors(rows.length);

  rows.forEach((item, i) => {
    const val = Number(item.value) || 0;
    const pct = total > 0 ? Math.round((val / total) * 100) : 0;
    legend.appendChild(
      el('div', { className: 'chart-legend__item' },
        el('span', {
          className: 'chart-legend__swatch',
          style: { background: colors[i] },
        }),
        el('span', { className: 'chart-legend__label' }, item.label),
        el('span', { className: 'chart-legend__value' }, `${val.toLocaleString('en-US')} (${pct}%)`),
      ),
    );
  });

  shell.onResize(() => {
    const surface = setupCanvas(shell.wrap, shell.canvas, CHART_HEIGHT_DEFAULT);
    if (!surface) return;
    const { ctx, width, height } = surface;

    ctx.clearRect(0, 0, width, height);

    const cx = width * 0.5;
    const cy = height * 0.48;
    const outerR = Math.min(width, height) * 0.36;
    const innerR = outerR * 0.58;
    let angle = -Math.PI / 2;

    for (let i = 0; i < rows.length; i++) {
      const val = Number(rows[i].value) || 0;
      const slice = (val / total) * Math.PI * 2;
      const end = angle + slice;
      drawDonutSlice(ctx, cx, cy, outerR, innerR, angle, end, colors[i]);
      angle = end;
    }

    ctx.fillStyle = chartColor('--text-main');
    ctx.font = '600 15px var(--font-family), system-ui, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(total.toLocaleString('en-US'), cx, cy - 1);
    ctx.fillStyle = chartColor('--text-muted');
    ctx.font = '11px var(--font-family), system-ui, sans-serif';
    ctx.fillText('total', cx, cy + 14);
  });

  return { destroy: () => shell.destroy() };
}
