import { adminKpiAccentValueClass } from '@/lib/admin_metric_tone';
import { cn } from '@/lib/utils';
import { SummaryBand } from '@/shell/ui_bands';

export type LandersListSummaryBoxProps = {
  className?: string;
  filteredTotal: number;
  filtersActive: boolean;
  total: number;
};

export function LandersListSummaryBox({
  className,
  filteredTotal,
  filtersActive,
  total,
}: LandersListSummaryBoxProps) {
  const scopeLabel = filtersActive ? `${filteredTotal} filtered` : `${total} landers`;

  return (
    <SummaryBand className={className}>
      <p className="m-0 shrink-0 whitespace-nowrap text-xs leading-none text-muted-foreground">
        <span className={cn('font-bold', adminKpiAccentValueClass[1])}>{scopeLabel}</span>
        {filtersActive && filteredTotal !== total ? <span>{` of ${total}`}</span> : null}
      </p>
    </SummaryBand>
  );
}
