import { rateBenchmarkToneClass } from '@/domains/campaigns/list/campaign_list_rate_tone';
import { adminMetricApprovedClass, adminMetricConversionClass } from '@/lib/admin_metric_tone';
import { cn } from '@/lib/utils';

export function tableCellClass(
  isZero?: boolean,
  extra?: string,
  emphasis?: 'primary' | 'secondary' | 'conversion' | 'approved'
): string {
  return cn(
    'font-numeric num',
    isZero && 'text-muted-foreground/60',
    emphasis === 'primary' && !isZero && 'font-semibold text-foreground',
    emphasis === 'secondary' && !isZero && 'text-muted-foreground',
    emphasis === 'conversion' && !isZero && adminMetricConversionClass,
    emphasis === 'approved' && !isZero && adminMetricApprovedClass,
    extra
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
