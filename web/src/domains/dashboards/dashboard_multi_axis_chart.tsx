import { format, isValid, parseISO } from 'date-fns';
import { useEffect, useMemo, useState } from 'react';
import {
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import type { DashboardSeriesPoint } from '@/domains/dashboards/buyer_dashboard_types';
import {
  buildDateAxisTicks,
  buildMoneyAxisScale,
  buildVolumeAxisScale,
  formatUsdAxisTick,
  formatUsdTooltip,
  formatVolumeAxisTick,
} from '@/domains/dashboards/dashboard_chart_scale';
import {
  DASHBOARD_CHART_SERIES_STYLES,
  type DashboardMetricId,
} from '@/domains/dashboards/dashboard_metrics';
import { displayCount } from '@/lib/display';
import { EmptyState } from '@/shell/empty_state';
import { cn } from '@/lib/utils';

export type DashboardMultiAxisChartProps = {
  series: DashboardSeriesPoint[];
  chartMetricIds: DashboardMetricId[];
  className?: string;
};

type ChartRow = {
  label: string;
  clicks: number;
  conversions: number;
  cost_micro: number;
  revenue_micro: number;
  profit_micro: number;
};

function toChartNumber(value: unknown): number {
  const parsed = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatVolumeTooltip(value: number): string {
  return displayCount(value);
}

function formatChartDate(label: string): string {
  const trimmed = label.trim();
  if (!trimmed) {
    return '';
  }
  const date = parseISO(trimmed);
  if (!isValid(date)) {
    return trimmed;
  }
  return format(date, 'dd MMMM');
}

function volumeScale(rows: ChartRow[]) {
  let max = 0;
  for (const row of rows) {
    max = Math.max(max, row.clicks, row.conversions);
  }
  return buildVolumeAxisScale(max);
}

function moneyScale(rows: ChartRow[]) {
  let min = 0;
  let max = 0;
  for (const row of rows) {
    for (const value of [row.cost_micro, row.revenue_micro, row.profit_micro]) {
      min = Math.min(min, value);
      max = Math.max(max, value);
    }
  }
  return buildMoneyAxisScale(min, max);
}

function ChartTooltipContent({
  active,
  payload,
  label,
  selected,
}: {
  active?: boolean;
  payload?: Array<{ dataKey?: string; value?: number; payload?: ChartRow }>;
  label?: string;
  selected: DashboardMetricId[];
}) {
  if (!active) {
    return null;
  }
  const row = payload?.[0]?.payload;
  if (!row) {
    return null;
  }

  const visible = DASHBOARD_CHART_SERIES_STYLES.filter((metric) => selected.includes(metric.id));
  const volumeEntries = visible.filter((metric) => metric.axis === 'volume');
  const moneyEntries = visible.filter((metric) => metric.axis === 'money');

  const renderSection = (
    title: string,
    metrics: typeof DASHBOARD_CHART_SERIES_STYLES,
  ) => {
    if (metrics.length === 0) {
      return null;
    }
    return (
      <div className="grid gap-1">
        <p className="font-numeric text-admin-mini tracking-wide text-muted-foreground">
          {title}
        </p>
        {metrics.map((metric) => {
          const value = row[metric.seriesKey as keyof ChartRow] as number;
          const formatted =
            metric.axis === 'money' ? formatUsdTooltip(value) : formatVolumeTooltip(value);
          return (
            <div key={metric.id} className="flex items-center justify-between gap-4">
              <span className="flex items-center gap-2 text-muted-foreground">
                <span
                  className="inline-block h-2 w-2 shrink-0 rounded-full"
                  style={{ backgroundColor: metric.stroke }}
                />
                {metric.label}
              </span>
              <span className="font-numeric tabular-nums text-foreground">{formatted}</span>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="ui-surface-raised grid min-w-[12rem] gap-2 rounded-xl border border-border/60 px-3 py-2 text-xs shadow-sm">
      <p className="font-medium text-foreground">{formatChartDate(label ?? row.label)}</p>
      {renderSection('Volume', volumeEntries)}
      {renderSection('USD', moneyEntries)}
    </div>
  );
}

const axisLineStyle = { stroke: 'hsl(var(--border) / 0.55)' };
const chartLineStroke = 'hsl(var(--foreground) / 0.36)';
const chartAxisLabelColor = 'hsl(var(--muted-foreground))';

const chartLineDash: Partial<Record<DashboardMetricId, string>> = {
  clicks: undefined,
  conversions: '5 4',
  cost: '7 4',
  revenue: '3 5',
  profit: '2 4',
};

function ChartLegendContent({
  metrics,
  activeIds,
  onToggle,
}: {
  metrics: typeof DASHBOARD_CHART_SERIES_STYLES;
  activeIds: DashboardMetricId[];
  onToggle: (metricId: DashboardMetricId) => void;
}) {
  return (
    <ul className="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 px-2 pt-2 text-xs">
      {metrics.map((metric) => {
        const active = activeIds.includes(metric.id);
        return (
          <li key={metric.id}>
            <button
              type="button"
              className={cn(
                'flex items-center gap-1.5 rounded-full px-1 py-0.5 transition-opacity focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-zinc-950',
                !active && 'opacity-45',
              )}
              aria-pressed={active}
              onClick={() => onToggle(metric.id)}
            >
              <span
                className="inline-block h-2 w-4 shrink-0 rounded-full"
                style={{ backgroundColor: metric.stroke }}
                aria-hidden
              />
              <span className={cn('text-muted-foreground', active && 'text-foreground')}>
                {metric.label}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}

export function DashboardMultiAxisChart({ series, chartMetricIds, className }: DashboardMultiAxisChartProps) {
  const [activeMetricIds, setActiveMetricIds] = useState<DashboardMetricId[]>(chartMetricIds);

  useEffect(() => {
    setActiveMetricIds(chartMetricIds);
  }, [chartMetricIds]);

  const configuredMetrics = useMemo(
    () => DASHBOARD_CHART_SERIES_STYLES.filter((metric) => chartMetricIds.includes(metric.id)),
    [chartMetricIds],
  );

  const visibleMetrics = useMemo(
    () => configuredMetrics.filter((metric) => activeMetricIds.includes(metric.id)),
    [activeMetricIds, configuredMetrics],
  );

  function toggleChartMetric(metricId: DashboardMetricId) {
    setActiveMetricIds((current) => {
      if (current.includes(metricId)) {
        if (current.length <= 1) {
          return current;
        }
        return current.filter((id) => id !== metricId);
      }
      return [...current, metricId];
    });
  }

  const chartRows = useMemo((): ChartRow[] => {
    return series.map((point) => {
      const costMicro = toChartNumber(point.spend_micro ?? point.spend_micros);
      const revenueMicro = toChartNumber(point.revenue_micro);
      const profitMicro = toChartNumber(point.profit_micro ?? revenueMicro - costMicro);
      return {
        label: point.label?.trim() ?? '',
        clicks: toChartNumber(point.clicks),
        conversions: toChartNumber(point.conversions),
        cost_micro: costMicro,
        revenue_micro: revenueMicro,
        profit_micro: profitMicro,
      };
    });
  }, [series]);

  const showVolumeAxis = visibleMetrics.some((metric) => metric.axis === 'volume');
  const showMoneyAxis = visibleMetrics.some((metric) => metric.axis === 'money');

  const filteredRows = chartRows;
  const volumeYScale = useMemo(() => volumeScale(filteredRows), [filteredRows]);
  const moneyYScale = useMemo(() => moneyScale(filteredRows), [filteredRows]);
  const dateAxisTicks = useMemo(
    () => buildDateAxisTicks(chartRows.map((row) => row.label)),
    [chartRows],
  );

  if (series.length === 0) {
    return (
      <section
        aria-label="Performance chart"
        className={cn('ui-surface-raised overflow-hidden p-3', className)}
      >
        <EmptyState
          className="border-0 bg-transparent py-10 shadow-none"
          description="Try a wider period or confirm the customer has traffic in this range."
          variant="no-results"
        />
      </section>
    );
  }

  return (
    <section
      aria-label="Performance chart"
      className={cn('ui-surface-raised overflow-hidden p-3', className)}
    >
      <div className="rounded-xl border border-border bg-card/25 p-2 sm:p-3">
        <div className="h-[min(30rem,52vh)] min-h-[300px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart
              data={chartRows}
              margin={{ top: 32, right: 12, left: 4, bottom: 36 }}
            >
              <CartesianGrid
                stroke="hsl(var(--border) / 0.28)"
                syncWithTicks
              />
              <XAxis
                xAxisId="dateTop"
                orientation="top"
                dataKey="label"
                ticks={dateAxisTicks}
                tick={{ fill: 'hsl(var(--muted-foreground))', fontSize: 10 }}
                tickLine={false}
                axisLine={axisLineStyle}
                tickFormatter={formatChartDate}
                interval={0}
                minTickGap={0}
              />
              <XAxis
                xAxisId="dateBottom"
                dataKey="label"
                ticks={dateAxisTicks}
                angle={-35}
                tick={{ fill: 'hsl(var(--muted-foreground))', fontSize: 10, textAnchor: 'end' }}
                tickLine={false}
                axisLine={axisLineStyle}
                tickFormatter={formatChartDate}
                interval={0}
                minTickGap={0}
                height={48}
              />
              {showVolumeAxis ? (
                <YAxis
                  yAxisId="volume"
                  domain={volumeYScale.domain}
                  ticks={volumeYScale.ticks}
                  allowDecimals={false}
                  tick={{
                    fill: chartAxisLabelColor,
                    fontSize: 10,
                    fontFamily: 'var(--font-numeric)',
                  }}
                  tickLine={false}
                  axisLine={{ stroke: 'hsl(var(--border) / 0.45)' }}
                  tickFormatter={(value) => formatVolumeAxisTick(Number(value))}
                  width={58}
                  label={{
                    value: 'Volume',
                    angle: -90,
                    position: 'insideLeft',
                    offset: 12,
                    style: { fill: chartAxisLabelColor, fontSize: 10 },
                  }}
                />
              ) : null}
              {showMoneyAxis ? (
                <YAxis
                  yAxisId="money"
                  orientation="right"
                  domain={moneyYScale.domain}
                  ticks={moneyYScale.ticks}
                  allowDecimals={false}
                  tick={{
                    fill: chartAxisLabelColor,
                    fontSize: 10,
                    fontFamily: 'var(--font-numeric)',
                  }}
                  tickLine={false}
                  axisLine={{ stroke: 'hsl(var(--border) / 0.45)' }}
                  tickFormatter={(value) => formatUsdAxisTick(Number(value))}
                  width={56}
                  label={{
                    value: 'USD',
                    angle: 90,
                    position: 'insideRight',
                    offset: 12,
                    style: { fill: chartAxisLabelColor, fontSize: 10 },
                  }}
                />
              ) : null}
              <Tooltip
                shared
                trigger="hover"
                cursor={{ stroke: 'hsl(var(--foreground) / 0.22)', strokeWidth: 1 }}
                content={<ChartTooltipContent selected={activeMetricIds} />}
                isAnimationActive={false}
              />
              {visibleMetrics.map((metric) => (
                <Line
                  key={metric.id}
                  xAxisId="dateBottom"
                  yAxisId={metric.axis === 'money' ? 'money' : 'volume'}
                  type="monotone"
                  dataKey={metric.seriesKey}
                  stroke={chartLineStroke}
                  strokeWidth={metric.axis === 'money' ? 1.75 : 2}
                  strokeDasharray={chartLineDash[metric.id]}
                  dot={false}
                  isAnimationActive={false}
                  activeDot={{
                    r: 3,
                    strokeWidth: 1.5,
                    stroke: chartLineStroke,
                    fill: 'hsl(var(--background))',
                  }}
                />
              ))}
            </ComposedChart>
          </ResponsiveContainer>
        </div>
        <ChartLegendContent
          activeIds={activeMetricIds}
          metrics={configuredMetrics}
          onToggle={toggleChartMetric}
        />
      </div>
    </section>
  );
}
