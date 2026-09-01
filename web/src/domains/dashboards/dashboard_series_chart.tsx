import { useId, useMemo } from 'react';

import { PanelSection } from '@/components/system/stat_panel';
import type { DashboardSeriesPoint } from '@/domains/dashboards/buyer_dashboard_types';
import { displayCount, displayMicro } from '@/lib/display';
import { cn } from '@/lib/utils';

export type DashboardSeriesChartProps = {
  series: DashboardSeriesPoint[];
  className?: string;
};

type ChartSeries = {
  id: string;
  label: string;
  strokeClass: string;
  values: number[];
  format: (value: number) => string;
};

const CHART_HEIGHT = 220;
const CHART_PAD = { top: 12, right: 12, bottom: 28, left: 44 };

function seriesMax(values: number[]): number {
  let max = 0;
  for (const value of values) {
    if (value > max) {
      max = value;
    }
  }
  return max;
}

function buildPolyline(
  values: number[],
  width: number,
  height: number,
  maxValue: number,
): string {
  if (values.length === 0 || maxValue <= 0) {
    return '';
  }
  const stepX = values.length > 1 ? width / (values.length - 1) : 0;
  return values
    .map((value, index) => {
      const x = values.length > 1 ? index * stepX : width / 2;
      const y = height - (value / maxValue) * height;
      return `${x},${y}`;
    })
    .join(' ');
}

export function DashboardSeriesChart({ series, className }: DashboardSeriesChartProps) {
  const gradientId = useId();

  const chart = useMemo(() => {
    const labels = series.map((point) => point.label?.trim() ?? '');
    const impressions = series.map((point) => point.impressions ?? 0);
    const blocks = series.map((point) => point.blocks ?? 0);
    const spend = series.map((point) => point.spend_micros ?? 0);

    const chartSeries: ChartSeries[] = [
      {
        id: 'impressions',
        label: 'Impressions',
        strokeClass: 'stroke-chart-1',
        values: impressions,
        format: (value) => displayCount(value),
      },
      {
        id: 'blocks',
        label: 'Blocks',
        strokeClass: 'stroke-chart-3',
        values: blocks,
        format: (value) => displayCount(value),
      },
      {
        id: 'spend',
        label: 'Spend (micro)',
        strokeClass: 'stroke-chart-5',
        values: spend,
        format: (value) => displayMicro(value),
      },
    ];

    const innerWidth = 640;
    const innerHeight = CHART_HEIGHT - CHART_PAD.top - CHART_PAD.bottom;

    return {
      labels,
      chartSeries,
      viewBox: `0 0 ${innerWidth + CHART_PAD.left + CHART_PAD.right} ${CHART_HEIGHT}`,
      innerWidth,
      innerHeight,
      paths: chartSeries.map((item) =>
        buildPolyline(item.values, innerWidth, innerHeight, seriesMax(item.values) || 1),
      ),
    };
  }, [series]);

  if (series.length === 0) {
    return (
      <PanelSection className={className} title="Delivery trend">
        <p className="px-5 py-8 text-sm text-muted-foreground">No series data for this range.</p>
      </PanelSection>
    );
  }

  return (
    <PanelSection className={className} title="Delivery trend">
      <div className="px-2 pb-4 pt-2 sm:px-4">
        <svg
          aria-label="Delivery trend chart"
          className="h-auto w-full text-muted-foreground"
          role="img"
          viewBox={chart.viewBox}
        >
          <defs>
            <linearGradient id={gradientId} x1="0" x2="0" y1="0" y2="1">
              <stop offset="0%" stopColor="hsl(var(--chart-1) / 0.12)" />
              <stop offset="100%" stopColor="hsl(var(--chart-1) / 0)" />
            </linearGradient>
          </defs>
          <g transform={`translate(${CHART_PAD.left} ${CHART_PAD.top})`}>
            {[0, 0.25, 0.5, 0.75, 1].map((tick) => {
              const y = chart.innerHeight * (1 - tick);
              const label = `${Math.round(tick * 100)}%`;
              return (
                <g key={tick}>
                  <line
                    className="stroke-border/50"
                    x1={0}
                    x2={chart.innerWidth}
                    y1={y}
                    y2={y}
                  />
                  <text className="fill-muted-foreground text-[10px] tabular-nums" x={-8} y={y + 3} textAnchor="end">
                    {label}
                  </text>
                </g>
              );
            })}
            {chart.paths[0] ? (
              <polygon
                fill={`url(#${gradientId})`}
                points={`0,${chart.innerHeight} ${chart.paths[0]} ${chart.innerWidth},${chart.innerHeight}`}
              />
            ) : null}
            {chart.chartSeries.map((item, index) => (
              <polyline
                key={item.id}
                className={cn('fill-none', item.strokeClass)}
                points={chart.paths[index]}
                strokeWidth={1.75}
                vectorEffect="non-scaling-stroke"
              />
            ))}
            {chart.labels.map((label, index) => {
              if (chart.labels.length > 12 && index % 2 !== 0) {
                return null;
              }
              const stepX =
                chart.labels.length > 1 ? chart.innerWidth / (chart.labels.length - 1) : 0;
              const x = chart.labels.length > 1 ? index * stepX : chart.innerWidth / 2;
              return (
                <text
                  key={`${label}-${index}`}
                  className="fill-muted-foreground text-[10px]"
                  textAnchor="middle"
                  x={x}
                  y={chart.innerHeight + 18}
                >
                  {label}
                </text>
              );
            })}
          </g>
        </svg>
        <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 px-2 pt-2">
          {chart.chartSeries.map((item) => {
            const total = item.values.reduce((sum, value) => sum + value, 0);
            return (
              <div key={item.id} className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className={cn('h-0.5 w-4 rounded-full', item.strokeClass.replace('stroke-', 'bg-'))} />
                <span>{item.label}</span>
                <span className="tabular-nums text-foreground">{item.format(total)}</span>
              </div>
            );
          })}
        </div>
      </div>
    </PanelSection>
  );
}
