import { el, replaceChildren } from '../lib/dom.js';
import {
  chartColor,
  chartPad,
  createChartShell,
  remPx,
  setupCanvas,
} from './canvas_util.js';

/**
 * Mount a simple canvas bar chart for categorical metrics.
 *
 * @param {HTMLElement} container
 * @param {Array<{ label: string, value: number }>} items
 * @param {string} [ariaLabel]
 * @returns {{ destroy: () => void }}
 */
export function mountBarChart(container, items, ariaLabel = 'Bar chart') {
  const rows = (items ?? []).filter((item) => item && item.label != null);
  if (rows.length === 0) {
    replaceChildren(container, el('p', { className: 'text-muted text-sm' }, 'No data to chart.'));
    return { destroy: () => container.replaceChildren() };
  }

  let maxVal = 0;
  for (let i = 0; i < rows.length; i++) {
    const v = Number(rows[i].value) || 0;
    if (v > maxVal) maxVal = v;
  }
  if (maxVal <= 0) maxVal = 1;

  const shell = createChartShell(container, ariaLabel);
  const PAD = chartPad({ left: 1, right: 1, top: 1, bottom: 2.25 });

  shell.onResize(() => {
    const cssHeight = Math.max(shell.wrap.clientHeight, remPx(5));
    const surface = setupCanvas(shell.wrap, shell.canvas, cssHeight);
    if (!surface) return;
    const { ctx, width, height } = surface;

    const plotW = width - PAD.left - PAD.right;
    const plotH = height - PAD.top - PAD.bottom;
    const barGap = remPx(0.75);
    const barW = Math.max(remPx(1), (plotW - barGap * (rows.length - 1)) / rows.length);

    ctx.clearRect(0, 0, width, height);
    const accent = chartColor('--accent');
    const muted = chartColor('--text-muted');
    const border = chartColor('--border-color');

    ctx.strokeStyle = border;
    ctx.lineWidth = 1;
    ctx.strokeRect(PAD.left, PAD.top, plotW, plotH);

    ctx.font = 'var(--text-xs) var(--font-family), system-ui, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    const labelY = height - PAD.bottom + remPx(0.125);

    for (let i = 0; i < rows.length; i++) {
      const val = Number(rows[i].value) || 0;
      const barH = (val / maxVal) * plotH;
      const x = PAD.left + i * (barW + barGap);
      const y = PAD.top + plotH - barH;

      ctx.fillStyle = accent;
      ctx.fillRect(x, y, barW, barH);

      ctx.fillStyle = muted;
      const label = rows[i].label.length > 10 ? `${rows[i].label.slice(0, 9)}…` : rows[i].label;
      ctx.fillText(label, x + barW / 2, labelY);
    }
  });

  return { destroy: () => shell.destroy() };
}
