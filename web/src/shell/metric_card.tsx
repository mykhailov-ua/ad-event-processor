import { ArrowDown, ArrowUp } from 'lucide-react';

import {
  adminKpiAccentSurfaceClass,
  adminKpiAccentTopBarClass,
  adminKpiAccentValueClass,
  adminMetricDeltaNegativeClass,
  adminMetricDeltaPositiveClass,
  type AdminKpiAccent,
} from '@/lib/admin_metric_tone';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type MetricCardProps = {
  label: string;
  value: string;
  deltaPct?: number | null;
  accent?: AdminKpiAccent;
  className?: string;
};

export function MetricCard({ label, value, deltaPct, accent, className }: MetricCardProps) {
  const delta = deltaPct ?? null;
  const showDelta = delta != null && Number.isFinite(delta);
  const positive = showDelta && delta >= 0;

  return (
    <div
      className={cn(
        'grid gap-2 border border-border bg-card p-4',
        adminKit.panelRadius,
        accent ? adminKpiAccentSurfaceClass[accent] : null,
        accent ? adminKpiAccentTopBarClass[accent] : null,
        className
      )}
    >
      <div className="grid grid-cols-[1fr_auto] items-start gap-2">
        <p className="m-0 text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground">
          {label}
        </p>
        {showDelta ? (
          <span
            className={cn(
              'inline-flex items-center gap-0.5 font-numeric text-[11px]',
              positive ? adminMetricDeltaPositiveClass : adminMetricDeltaNegativeClass
            )}
          >
            {positive ? (
              <ArrowUp aria-hidden className="h-3 w-3" />
            ) : (
              <ArrowDown aria-hidden className="h-3 w-3" />
            )}
            {positive ? '+' : ''}
            {delta.toFixed(0)}%
          </span>
        ) : null}
      </div>
      <p
        className={cn(
          'm-0 font-numeric text-2xl leading-none',
          accent ? adminKpiAccentValueClass[accent] : 'text-foreground'
        )}
      >
        {value}
      </p>
    </div>
  );
}
