import { rateBenchmarkToneClass } from '@/domains/campaigns/list/campaign_list_rate_tone';
import { cn } from '@/lib/utils';

export function tableCellClass(
  isZero?: boolean,
  extra?: string,
  emphasis?: 'primary' | 'secondary' | 'conversion' | 'approved',
): string {
  return cn(
    'tabular-nums num',
    isZero && 'tabular-nums text-muted-foreground/60',
    emphasis === 'primary' && !isZero && 'tabular-nums font-semibold text-foreground',
    emphasis === 'secondary' && !isZero && 'tabular-nums text-muted-foreground',
    emphasis === 'conversion' && !isZero && 'tabular-nums font-bold text-indigo-600',
    emphasis === 'approved' && !isZero && 'tabular-nums font-semibold text-emerald-700',
    extra,
  );
}

export function RateMetricCell({
  children,
  isEmpty,
  ratePct,
}: {
  children: string;
  isEmpty: boolean;
  ratePct: number | null;
}) {
  return (
    <span className={tableCellClass(isEmpty, rateBenchmarkToneClass(isEmpty ? null : ratePct))}>
      {children}
    </span>
  );
}
