import { el } from '../lib/dom.js';
import { formatChartTick, formatChartAxisTime } from '../helpers/chart_format.js';
import uPlot from 'uplot';
import { withAlphaCached } from './chart_math.js';
import { chartColor, alphaCache } from './canvas_util.js';

export type ThemedAxesOpts = {
  rangeMs: number;
  formatY?: (value: number) => string;
};

/**
 * Shared uPlot axis styling from CSS tokens.
 */
export function themedAxes(opts: ThemedAxesOpts): uPlot.Axis[] {
  const muted = chartColor('--text-muted');
  const border = chartColor('--border-color');
  const formatY = opts.formatY ?? formatChartTick;
  return [
    {
      stroke: muted,
      grid: { stroke: border, width: 1 },
      ticks: { stroke: border },
      values: (_u, splits) => splits.map((v) => formatChartAxisTime(v * 1000, opts.rangeMs)),
    },
    {
      stroke: muted,
      grid: { stroke: border, width: 1 },
      ticks: { stroke: border },
      values: (_u, splits) => splits.map((v) => formatY(v)),
    },
  ];
}

/**
 * Area line series using theme accent color.
 */
export function themedAreaSeries(colorToken = '--accent'): uPlot.Series[] {
  const accent = chartColor(colorToken);
  return [
    {},
    {
      stroke: accent,
      width: 2,
      fill: withAlphaCached(accent, 0.22, alphaCache),
      points: { show: false },
    },
  ];
}

/**
 * Cursor with crosshair and hover point.
 */
export function themedCursor(): uPlot.Cursor {
  const accent = chartColor('--accent');
  const surface = chartColor('--bg-surface');
  return {
    show: true,
    focus: { prox: 20 },
    points: {
      show: true,
      size: 6,
      stroke: accent,
      fill: surface,
      width: 2,
    },
    drag: { x: false, y: false },
  };
}

/**
 * HTML tooltip hooks for uPlot.
 */
export function createTooltipHooks(
  root: HTMLElement,
  formatX: (tsSec: number) => string,
  formatY: (value: number) => string,
): uPlot.Hooks.Arrays {
  const tip = el('div', { className: 'uplot-tooltip' }) as HTMLDivElement;
  tip.hidden = true;
  root.appendChild(tip);

  return {
    setCursor: [
      (u) => {
        const idx = u.cursor.idx;
        if (idx == null) {
          tip.hidden = true;
          return;
        }
        const x = u.data[0][idx] as number;
        const y = u.data[1][idx] as number;
        tip.textContent = `${formatX(x)} · ${formatY(y)}`;
        tip.hidden = false;
        const left = u.cursor.left ?? 0;
        const top = u.cursor.top ?? 0;
        tip.style.transform = `translate(${Math.round(left + 12)}px, ${Math.round(top + 12)}px)`;
      },
    ],
  };
}
