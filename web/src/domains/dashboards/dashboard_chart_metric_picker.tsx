import type { DashboardMetricId } from '@/domains/dashboards/dashboard_metrics';
import { DASHBOARD_CHART_SERIES_STYLES } from '@/domains/dashboards/dashboard_metrics';
import { cn } from '@/lib/utils';

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
      <span className="text-admin-caption tracking-wide text-muted-foreground">Metrics</span>
      {DASHBOARD_CHART_SERIES_STYLES.map((metric) => {
        const active = selected.includes(metric.id);
        return (
          <button
            key={metric.id}
            type="button"
            aria-pressed={active}
            onClick={() => onToggle(metric.id)}
            className={cn(
              'rounded-sm border px-2.5 py-1 font-geist text-admin-caption font-medium transition-opacity',
              active ? 'opacity-100' : 'opacity-35 hover:opacity-60',
            )}
            style={{
              borderColor: metric.stroke,
              color: metric.stroke,
              backgroundColor: active ? metric.fill : 'transparent',
            }}
          >
            {metric.label}
          </button>
        );
      })}
    </div>
  );
}
