import { el, replaceChildren } from '../lib/dom.js';
import { formatChartTick, formatChartAxisTime } from '../helpers/chart_format.js';
import { copyMetricPoints, rangeMsFromHours } from '../helpers/ops_metric_series.js';
import type { MetricPoint } from '../helpers/ops_metric_series.js';
import uPlot from 'uplot';
import { SERIES_CAP } from './chart_math.js';
import type { ChartHandle } from './chart_types.js';
import {
  createTooltipHooks,
  themedAreaSeries,
  themedAxes,
  themedCursor,
} from './uplot_theme.js';

export const CHART_HEIGHT_METRIC = 200;

export type MetricChartOpts = {
  title: string;
  points?: MetricPoint[];
  value?: number;
  min?: number;
  max?: number;
  rangeHours?: number;
  color?: string;
  formatValue?: (value: number) => string;
};

export type MetricChartUpdate = {
  points?: MetricPoint[];
  value?: number;
  min?: number;
  max?: number;
  rangeHours?: number;
  color?: string;
};

export type MetricChartHandle = ChartHandle & {
  update: (next: MetricChartUpdate) => void;
};

/**
 * Mount a Grafana-style time-series area chart (uPlot).
 */
export function mountMetricChart(container: HTMLElement, opts: MetricChartOpts): MetricChartHandle {
  const title = opts.title ?? 'Metric';
  let colorToken = opts.color ?? '--accent';
  let yMinOpt = Number.isFinite(opts.min) ? (opts.min as number) : NaN;
  let yMaxOpt = Number.isFinite(opts.max) ? (opts.max as number) : NaN;
  let rangeMs = rangeMsFromHours(opts.rangeHours ?? 24);
  let fallbackValue = Number(opts.value) || 0;
  let pointsRef: MetricPoint[] | null = opts.points ?? null;
  const formatY = opts.formatValue ?? formatChartTick;

  const seriesTs = new Float64Array(SERIES_CAP);
  const seriesVal = new Float64Array(SERIES_CAP);
  const dataTs = new Float64Array(SERIES_CAP);

  const wrap = el('div', { className: 'chart-shell metric-chart-uplot' }) as HTMLDivElement;
  const root = el('div', {
    className: 'chart-root chart-root--clip metric-chart-uplot__root',
    role: 'img',
    'aria-label': title,
  }) as HTMLDivElement;
  wrap.appendChild(root);
  replaceChildren(container, wrap);

  let plot: uPlot | null = null;
  let ro: ResizeObserver | null = null;
  let lastWidth = 0;

  function chartWidth(): number {
    return Math.max(root.clientWidth || wrap.clientWidth || 320, 120);
  }

  function reloadData(): uPlot.AlignedData {
    const n = copyMetricPoints(pointsRef, fallbackValue, rangeMs, seriesTs, seriesVal);
    for (let i = 0; i < n; i++) {
      dataTs[i] = seriesTs[i] / 1000;
    }
    return [dataTs.subarray(0, n), seriesVal.subarray(0, n)];
  }

  const formatX = (tsSec: number) => formatChartAxisTime(tsSec * 1000, rangeMs);

  function buildOptions(width: number): uPlot.Options {
    return {
      width,
      height: CHART_HEIGHT_METRIC,
      title,
      cursor: themedCursor(),
      legend: { show: false },
      scales: {
        x: { time: true },
        y: {
          range: (_u, dataMin, dataMax) => {
            if (Number.isFinite(yMinOpt) && Number.isFinite(yMaxOpt)) {
              return [yMinOpt, yMaxOpt];
            }
            return uPlot.rangeNum(dataMin, dataMax, 0.1, true);
          },
        },
      },
      axes: themedAxes({ rangeMs, formatY }),
      series: themedAreaSeries(colorToken),
      hooks: createTooltipHooks(root, formatX, formatY),
    };
  }

  function render(recreate = false): void {
    const width = chartWidth();
    const data = reloadData();
    if (!plot || recreate) {
      plot?.destroy();
      plot = new uPlot(buildOptions(width), data, root);
      lastWidth = width;
      return;
    }
    if (width !== lastWidth) {
      plot.setSize({ width, height: CHART_HEIGHT_METRIC });
      lastWidth = width;
    }
    plot.setData(data, false);
  }

  ro = new ResizeObserver(() => render());
  ro.observe(wrap);
  render();

  return {
    update(next: MetricChartUpdate) {
      let recreate = false;
      if (next.color && next.color !== colorToken) {
        colorToken = next.color;
        recreate = true;
      }
      if (Number.isFinite(next.min)) yMinOpt = next.min as number;
      if (Number.isFinite(next.max)) yMaxOpt = next.max as number;
      if (next.rangeHours != null) {
        rangeMs = rangeMsFromHours(next.rangeHours);
        recreate = true;
      }
      if (next.value != null) fallbackValue = Number(next.value) || 0;
      if (next.points) pointsRef = next.points;
      if (Number.isFinite(next.min) || Number.isFinite(next.max)) recreate = true;
      render(recreate);
    },
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
 * Mount an empty metric chart placeholder.
 */
export function mountMetricChartEmpty(container: HTMLElement): ChartHandle {
  replaceChildren(container, el('p', { className: 'text-muted text-sm' }, 'No data to chart.'));
  return { destroy: () => container.replaceChildren() };
}
