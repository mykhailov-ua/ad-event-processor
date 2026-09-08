import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import type { DashboardSeriesPoint } from '@/domains/dashboards/buyer_dashboard_types';
import {
  buildDateAxisTicks,
  buildMoneyAxisScale,
  buildVolumeAxisScale,
} from '@/domains/dashboards/dashboard_chart_scale';
import {
  DashboardMultiAxisChartCanvas,
  type DashboardChartRow,
} from '@/domains/dashboards/dashboard_multi_axis_chart_canvas';
import {
  DASHBOARD_CHART_SERIES_STYLES,
  type DashboardMetricId,
} from '@/domains/dashboards/dashboard_metrics';
import { EmptyState } from '@/shell/empty_state';
import { Button } from '@/components/ui/button';
import { DashboardCard } from '@/domains/dashboards/dashboard_card';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type DashboardMultiAxisChartProps = {
  series: DashboardSeriesPoint[];
  chartMetricIds: DashboardMetricId[];
  className?: string;
};

function toChartNumber(value: unknown): number {
  const parsed = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function volumeScale(rows: DashboardChartRow[]) {
  let max = 0;
  for (const row of rows) {
    max = Math.max(max, row.clicks, row.conversions);
  }
  return buildVolumeAxisScale(max);
}

function moneyScale(rows: DashboardChartRow[]) {
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
        const isLastActive = active && activeIds.length <= 1;
        return (
          <li key={metric.id}>
            <Button
              aria-pressed={active}
              className={cn(
                'h-auto flex items-center gap-1.5 rounded-full px-1 py-0.5 shadow-none',
                !active && 'opacity-45'
              )}
              disabled={isLastActive}
              title={isLastActive ? 'At least one metric must stay visible' : undefined}
              type="button"
              variant="ghost"
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
            </Button>
          </li>
        );
      })}
    </ul>
  );
}

export function DashboardMultiAxisChart({
  series,
  chartMetricIds,
  className,
}: DashboardMultiAxisChartProps) {
  const [activeMetricIds, setActiveMetricIds] = useState<DashboardMetricId[]>(chartMetricIds);

  useEffect(() => {
    setActiveMetricIds(chartMetricIds);
  }, [chartMetricIds]);

  const configuredMetrics = useMemo(
    () => DASHBOARD_CHART_SERIES_STYLES.filter((metric) => chartMetricIds.includes(metric.id)),
    [chartMetricIds]
  );

  const visibleMetrics = useMemo(
    () => configuredMetrics.filter((metric) => activeMetricIds.includes(metric.id)),
    [activeMetricIds, configuredMetrics]
  );

  function toggleChartMetric(metricId: DashboardMetricId) {
    setActiveMetricIds((current) => {
      if (current.includes(metricId)) {
        if (current.length <= 1) {
          toast.message('At least one metric must stay visible');
          return current;
        }
        return current.filter((id) => id !== metricId);
      }
      return [...current, metricId];
    });
  }

  const chartRows = useMemo((): DashboardChartRow[] => {
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

  const volumeYScale = useMemo(() => volumeScale(chartRows), [chartRows]);
  const moneyYScale = useMemo(() => moneyScale(chartRows), [chartRows]);
  const dateAxisTicks = useMemo(
    () => buildDateAxisTicks(chartRows.map((row) => row.label)),
    [chartRows]
  );

  if (series.length === 0) {
    return (
      <DashboardCard bodyClassName="py-8" className={className} title="Performance">
        <EmptyState
          className="border-0 bg-transparent py-2 shadow-none"
          description="Try a wider period or confirm the customer has traffic in this range."
          variant="no-results"
        />
      </DashboardCard>
    );
  }

  return (
    <DashboardCard
      bodyClassName="px-5 pb-4 pt-2"
      className={cn('overflow-hidden', className)}
      headerClassName="border-b-0 pb-0 pt-3"
      title="Performance"
    >
      <div className={cn('border border-border bg-muted/10 px-2 py-3 sm:px-3', adminKit.panelRadius)}>
        <DashboardMultiAxisChartCanvas
          activeMetricIds={activeMetricIds}
          chartRows={chartRows}
          dateAxisTicks={dateAxisTicks}
          moneyYScale={moneyYScale}
          volumeYScale={volumeYScale}
          visibleMetrics={visibleMetrics}
        />
        <ChartLegendContent
          activeIds={activeMetricIds}
          metrics={configuredMetrics}
          onToggle={toggleChartMetric}
        />
      </div>
    </DashboardCard>
  );
}
