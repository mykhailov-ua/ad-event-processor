import { budgetUtilizationPercent, budgetUtilizationRatio } from '@/lib/campaign_budget';
import { displayMoneyDecimal } from '@/lib/display';
import { cn } from '@/lib/utils';

export type CampaignSpendCellProps = {
  budgetLimit?: string;
  budgetLimitDisplay?: string;
  currentSpend?: string;
  currentSpendDisplay?: string;
};

export function CampaignSpendCell({
  budgetLimit,
  budgetLimitDisplay,
  currentSpend,
  currentSpendDisplay,
}: CampaignSpendCellProps) {
  const spendLabel = displayMoneyDecimal(currentSpend, currentSpendDisplay);
  const budgetLabel = displayMoneyDecimal(budgetLimit, budgetLimitDisplay);
  const ratio = budgetUtilizationRatio(currentSpend, budgetLimit);
  const percent = budgetUtilizationPercent(currentSpend, budgetLimit);
  const utilizationTitle =
    ratio != null && percent != null ? `${percent}% of ${budgetLabel} budget` : undefined;

  return (
    <div className="flex min-w-0 items-center gap-2" title={utilizationTitle}>
      <span className="shrink-0 tabular-nums">{spendLabel}</span>
      {ratio != null && percent != null ? (
        <div
          aria-hidden
          className="h-1 min-w-[2.5rem] flex-1 overflow-hidden rounded-full bg-muted"
        >
          <div
            className={cn(
              'h-full rounded-full transition-[width]',
              ratio >= 0.9
                ? 'bg-destructive'
                : ratio >= 0.7
                  ? 'bg-secondary'
                  : 'bg-primary',
            )}
            style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
          />
        </div>
      ) : null}
    </div>
  );
}
