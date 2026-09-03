import { rateBenchmarkToneClass } from '@/domains/campaigns/list/campaign_list_rate_tone';
import { cn } from '@/lib/utils';

export function tableCellClass(
  isZero?: boolean,
  extra?: string,
  emphasis?: 'primary' | 'secondary' | 'conversion',
): string {
  return cn(
    'tabular-nums num',
    isZero && 'tabular-nums text-zinc-400 dark:text-zinc-600',
    emphasis === 'primary' && !isZero && 'tabular-nums font-semibold text-zinc-900 dark:text-zinc-50',
    emphasis === 'secondary' && !isZero && 'tabular-nums text-zinc-600 dark:text-zinc-400',
    emphasis === 'conversion' && !isZero && 'tabular-nums font-bold text-indigo-600 dark:text-indigo-400',
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
