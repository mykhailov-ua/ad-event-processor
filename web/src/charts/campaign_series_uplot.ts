import { el, replaceChildren } from '../lib/dom.js';
import { formatChartTick, formatChartAxisTime } from '../helpers/chart_format.js';
import { seriesFromHourly } from '../helpers/chart_pool.js';
import type { HourlyMetricRow } from '../helpers/chart_pool.js';
import uPlot from 'uplot';
import type { ChartHandle } from './chart_types.js';
import type { SpendCurvePoint } from './campaign_chart_types.js';
import {
  createTooltipHooks,
  themedAreaSeries,
  themedAxes,
  themedCursor,
} from './uplot_theme.js';

export const CHART_HEIGHT_CAMPAIGN = 220;

export type CampaignChartOptions = {
  field?: string;
  label?: string;
};

function seriesRangeMs(x: Float64Array, length: number): number {
  if (length < 2) return 24 * 60 * 60 * 1000;
  return Math.max((x[length - 1] - x[0]) * 1000, 60 * 60 * 1000);
}

/**
 * Mount uPlot time-series for campaign hourly metrics.
 */
export function mountCampaignSeriesChart(
  container: HTMLElement,
  hourly: HourlyMetricRow[] | null | undefined,
  options: CampaignChartOptions = {},
): ChartHandle {
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
    replaceChildren(container, el('p', { className: 'text-muted text-sm' }, 'No series points'));
    return { destroy: () => container.replaceChildren() };
  }

  const rangeMs = seriesRangeMs(x, length);
  const data: uPlot.AlignedData = [x.subarray(0, length), y.subarray(0, length)];

  const wrap = el('div', { className: 'chart-shell metric-chart-uplot' }) as HTMLDivElement;
  const root = el('div', {
    className: 'chart-root chart-root--clip metric-chart-uplot__root',
    role: 'img',
    'aria-label': `${label} chart`,
  }) as HTMLDivElement;
  wrap.appendChild(root);
  replaceChildren(container, wrap);

  let plot: uPlot | null = null;
  let ro: ResizeObserver | null = null;
  let lastWidth = 0;

  const formatX = (tsSec: number) => formatChartAxisTime(tsSec * 1000, rangeMs);

  function chartWidth(): number {
    return Math.max(root.clientWidth || wrap.clientWidth || 320, 120);
  }

  function buildOptions(width: number): uPlot.Options {
    return {
      width,
      height: CHART_HEIGHT_CAMPAIGN,
      title: label,
      cursor: themedCursor(),
      legend: { show: false },
      scales: {
        x: { time: true },
        y: { range: (_u, dataMin, dataMax) => uPlot.rangeNum(dataMin, dataMax, 0.1, true) },
      },
      axes: themedAxes({ rangeMs, formatY: formatChartTick }),
      series: themedAreaSeries('--accent'),
      hooks: createTooltipHooks(root, formatX, formatChartTick),
    };
  }

  function render(recreate = false): void {
    const width = chartWidth();
    if (!plot || recreate) {
      plot?.destroy();
      plot = new uPlot(buildOptions(width), data, root);
      lastWidth = width;
      return;
    }
    if (width !== lastWidth) {
      plot.setSize({ width, height: CHART_HEIGHT_CAMPAIGN });
      lastWidth = width;
    }
    plot.setData(data, false);
  }

  ro = new ResizeObserver(() => render());
  ro.observe(wrap);
  render();

  return {
    destroy() {
      ro?.disconnect();
      ro = null;
      plot?.destroy();
      plot = null;
      container.replaceChildren();
    },
  };
}

/**
 * Mount forecast spend-curve line chart.
 */
export function mountSpendCurveChart(
  container: HTMLElement,
  curve: SpendCurvePoint[] | null | undefined,
  field: 'impressions' | 'spend_micro' = 'impressions',
): ChartHandle {
  const hourly: HourlyMetricRow[] = [];
  const src = curve ?? [];
  for (let i = 0; i < src.length; i++) {
    const point = src[i];
    hourly.push({
      hour: point.hour,
      [field]: Number(point[field]) || 0,
    });
  }
  const chartLabel = field === 'spend_micro' ? 'Projected spend' : 'Projected impressions';
  return mountCampaignSeriesChart(container, hourly, { field, label: chartLabel });
}
