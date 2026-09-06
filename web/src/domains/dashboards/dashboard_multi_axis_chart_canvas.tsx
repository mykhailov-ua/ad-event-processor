import { format, isValid, parseISO } from 'date-fns';
import { memo, useEffect, useRef, useState } from 'react';
import { CartesianGrid, ComposedChart, Line, Tooltip, XAxis, YAxis } from 'recharts';

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
import {
  markAdminPerf,
  measureAdminPerf,
  publishAdminPerfDuration,
} from '@/lib/perf/browser_marks';
import { cn } from '@/lib/utils';

// Hot chart leaf (frontend-hot-path.mdc regime F): Recharts owns series updates; parent passes snapshot rows.
// Optional ?admin_perf=1 marks via browser_marks.ts (Playwright perf tier, not tracker SLA).
export type DashboardChartRow = {
  label: string;
  clicks: number;
  conversions: number;
  cost_micro: number;
  revenue_micro: number;
  profit_micro: number;
};

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

function ChartTooltipContent({
  active,
  payload,
  label,
  selected,
}: {
  active?: boolean;
  payload?: Array<{ dataKey?: string; value?: number; payload?: DashboardChartRow }>;
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

  const renderSection = (title: string, metrics: typeof DASHBOARD_CHART_SERIES_STYLES) => {
    if (metrics.length === 0) {
      return null;
    }
    return (
      <div className="grid gap-1">
        <p className="font-numeric text-ui-mini tracking-wide text-muted-foreground">{title}</p>
        {metrics.map((metric) => {
          const value = row[metric.seriesKey as keyof DashboardChartRow] as number;
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
    <div className={cn('ui-surface-raised grid min-w-[12rem] gap-2 px-3 py-2 text-xs shadow-sm')}>
      <p className="font-medium text-foreground">{formatChartDate(label ?? row.label)}</p>
      {renderSection('Volume', volumeEntries)}
      {renderSection('USD', moneyEntries)}
    </div>
  );
}

function useRafCoalescedChartSize() {
  const ref = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const node = ref.current;
    if (!node) {
      return;
    }
    let frame = 0;
    const commit = () => {
      frame = 0;
      setSize({ width: node.clientWidth, height: node.clientHeight });
    };
    const schedule = () => {
      if (frame) {
        cancelAnimationFrame(frame);
      }
      frame = requestAnimationFrame(commit);
    };
    schedule();
    const observer = new ResizeObserver(schedule);
    observer.observe(node);
    return () => {
      observer.disconnect();
      if (frame) {
        cancelAnimationFrame(frame);
      }
    };
  }, []);

  return { ref, ...size };
}

export type DashboardMultiAxisChartCanvasProps = {
  chartRows: DashboardChartRow[];
  visibleMetrics: typeof DASHBOARD_CHART_SERIES_STYLES;
  activeMetricIds: DashboardMetricId[];
  volumeYScale: ReturnType<typeof buildVolumeAxisScale>;
  moneyYScale: ReturnType<typeof buildMoneyAxisScale>;
  dateAxisTicks: string[];
};

export const DashboardMultiAxisChartCanvas = memo(function DashboardMultiAxisChartCanvas({
  chartRows,
  visibleMetrics,
  activeMetricIds,
  volumeYScale,
  moneyYScale,
  dateAxisTicks,
}: DashboardMultiAxisChartCanvasProps) {
  const { ref, width, height } = useRafCoalescedChartSize();

  useEffect(() => {
    if (chartRows.length === 0) {
      return;
    }
    markAdminPerf('dashboard-chart-mount');
    const duration = measureAdminPerf('dashboard-chart-rows', 'dashboard-chart-mount');
    if (duration != null) {
      publishAdminPerfDuration('dashboard-chart-rows', duration);
    }
  }, [chartRows]);

  const showVolumeAxis = visibleMetrics.some((metric) => metric.axis === 'volume');
  const showMoneyAxis = visibleMetrics.some((metric) => metric.axis === 'money');

  return (
    <div ref={ref} className="h-[min(30rem,52vh)] min-h-[300px] w-full">
      {width > 0 && height > 0 ? (
        <ComposedChart
          data={chartRows}
          height={height}
          margin={{ top: 32, right: 12, left: 4, bottom: 36 }}
          width={width}
        >
          <CartesianGrid stroke="hsl(var(--border) / 0.28)" syncWithTicks />
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
      ) : null}
    </div>
  );
});
