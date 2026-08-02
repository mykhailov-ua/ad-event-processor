import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';
import { el } from '../lib/dom.js';
import { seriesFromHourly } from '../helpers/chart_pool.js';

const CHART_HEIGHT = 280;

/**
 * @param {HTMLElement} container
 * @param {Array<{ hour: string, impressions?: number }>} hourly
 * @param {string} [label]
 * @returns {{ destroy: () => void }}
 */
export function mountChart(container, hourly, label = 'Impressions') {
  let plot = null;
  let resizeTimer = null;
  let ro = null;

  function destroy() {
    if (resizeTimer) clearTimeout(resizeTimer);
    ro?.disconnect();
    plot?.destroy();
    plot = null;
  }

  if (!hourly?.length) {
    replaceChildren(container,
      el('div', { className: 'empty-state', style: { padding: '32px 16px' } },
        el('div', { className: 'empty-state__title' }, 'No data in period'),
        el('div', { className: 'empty-state__desc' }, 'Hourly metrics appear after traffic.'),
      ),
    );
    return { destroy: () => container.replaceChildren() };
  }

  const root = el('div', { className: 'chart-root', style: { width: '100%' } });
  replaceChildren(container, root);

  const { x, y, length } = seriesFromHourly(hourly, 'impressions');
  if (length === 0) {
    return { destroy: () => container.replaceChildren() };
  }

  const width = root.clientWidth || 640;
  const gridColor = getComputedStyle(document.documentElement)
    .getPropertyValue('--chart-grid').trim() || '#2a2f3a';
  const lineColor = getComputedStyle(document.documentElement)
    .getPropertyValue('--chart-line').trim() || '#7a73ff';

  plot = new uPlot({
    width,
    height: CHART_HEIGHT,
    series: [
      {},
      { label, stroke: lineColor, width: 2 },
    ],
    axes: [
      { stroke: gridColor, grid: { stroke: gridColor } },
      { stroke: gridColor, grid: { stroke: gridColor } },
    ],
    scales: { x: { time: true } },
  }, [x, y], root);

  ro = new ResizeObserver(() => {
    if (resizeTimer) clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
      if (plot && root) {
        plot.setSize({ width: root.clientWidth, height: CHART_HEIGHT });
      }
    }, 150);
  });
  ro.observe(root);

  return { destroy };
}

/**
 * @param {HTMLElement} node
 * @param {...(Node|string|null|undefined|false)} children
 */
function replaceChildren(node, ...children) {
  node.replaceChildren();
  for (const child of children) {
    if (child == null || child === false) continue;
    if (typeof child === 'string') node.appendChild(document.createTextNode(child));
    else node.appendChild(child);
  }
}
