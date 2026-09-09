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
};

export function MetricCard({ label, value, deltaPct, accent, }: MetricCardProps) {
  const delta = deltaPct ?? null;
  const showDelta = delta != null && Number.isFinite(delta);
  const positive = showDelta && delta >= 0;

  return (
    <div
     
    >
      <div >
        <p >
          {label}
        </p>
        {showDelta ? (
          <span
           
          >
            {positive ? (
              <ArrowUp aria-hidden  />
            ) : (
              <ArrowDown aria-hidden  />
            )}
            {positive ? '+' : ''}
            {delta.toFixed(0)}%
          </span>
        ) : null}
      </div>
      <p
       
      >
        {value}
      </p>
    </div>
  );
}
