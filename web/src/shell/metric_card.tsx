import { ArrowDown, ArrowUp } from 'lucide-react';

import { cn } from '@/lib/utils';

export type MetricCardProps = {
  label: string;
  value: string;
  deltaPct?: number | null;
  className?: string;
};

export function MetricCard({ label, value, deltaPct, className }: MetricCardProps) {
  const delta = deltaPct ?? null;
  const showDelta = delta != null && Number.isFinite(delta);
  const positive = showDelta && delta >= 0;

  return (
    <div className={cn('admin-metric-card', className)}>
      <div className="flex items-start justify-between gap-2">
        <p className="admin-metric-card__label">{label}</p>
        {showDelta ? (
          <span
            className={cn(
              'inline-flex items-center gap-0.5 text-[11px] font-semibold tabular-nums',
              positive ? 'text-emerald-600' : 'text-red-600',
            )}
          >
            {positive ? <ArrowUp aria-hidden className="h-3 w-3" /> : <ArrowDown aria-hidden className="h-3 w-3" />}
            {positive ? '+' : ''}
            {delta.toFixed(0)}%
          </span>
        ) : null}
      </div>
      <p className="admin-metric-card__value tabular-nums">{value}</p>
    </div>
  );
}
