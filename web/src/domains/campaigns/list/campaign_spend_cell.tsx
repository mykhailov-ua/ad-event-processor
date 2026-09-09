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
    <div  title={utilizationTitle}>
      <span >{spendLabel}</span>
      {ratio != null && percent != null ? (
        <div
          aria-hidden
         
        >
          <div
           
           
          />
        </div>
      ) : null}
    </div>
  );
}
