import type { DashboardMetricId } from '@/domains/dashboards/dashboard_metrics';
import { DASHBOARD_CHART_SERIES_STYLES } from '@/domains/dashboards/dashboard_metrics';
import { Button } from '@/components/ui/button';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

const CHART_TOKEN_CLASS: Record<
  (typeof DASHBOARD_CHART_SERIES_STYLES)[number]['chartToken'],
  { border: string; text: string; activeBg: string }
> = {
  1: {
    border: 'border-[hsl(var(--chart-1))]',
    text: 'text-[hsl(var(--chart-1))]',
    activeBg: 'bg-[hsl(var(--chart-1)/0.12)]',
  },
  2: {
    border: 'border-[hsl(var(--chart-2))]',
    text: 'text-[hsl(var(--chart-2))]',
    activeBg: 'bg-[hsl(var(--chart-2)/0.12)]',
  },
  3: {
    border: 'border-[hsl(var(--chart-3))]',
    text: 'text-[hsl(var(--chart-3))]',
    activeBg: 'bg-[hsl(var(--chart-3)/0.12)]',
  },
  4: {
    border: 'border-[hsl(var(--chart-4))]',
    text: 'text-[hsl(var(--chart-4))]',
    activeBg: 'bg-[hsl(var(--chart-4)/0.12)]',
  },
  5: {
    border: 'border-[hsl(var(--chart-5))]',
    text: 'text-[hsl(var(--chart-5))]',
    activeBg: 'bg-[hsl(var(--chart-5)/0.12)]',
  },
};

export type DashboardChartMetricPickerProps = {
  selected: DashboardMetricId[];
  onToggle: (id: DashboardMetricId) => void;
  className?: string;
};

export function DashboardChartMetricPicker({
  selected,
  onToggle,
  className,
}: DashboardChartMetricPickerProps) {
  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      <span className="text-ui-caption tracking-wide text-muted-foreground">Metrics</span>
      {DASHBOARD_CHART_SERIES_STYLES.map((metric) => {
        const active = selected.includes(metric.id);
        const tokenClass = CHART_TOKEN_CLASS[metric.chartToken];
        return (
          <Button
            key={metric.id}
            aria-pressed={active}
            className={cn(
              adminKit.controlHeight,
              'rounded-sm border px-2.5 text-ui-caption font-medium shadow-none',
              tokenClass.border,
              tokenClass.text,
              active
                ? cn(tokenClass.activeBg, 'opacity-100')
                : 'bg-transparent opacity-35 hover:opacity-60'
            )}
            type="button"
            variant="outline"
            onClick={() => onToggle(metric.id)}
          >
            {metric.label}
          </Button>
        );
      })}
    </div>
  );
}
