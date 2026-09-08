import {
  formatTableCount,
  formatTableMoneyFromMicro,
} from '@/domains/campaigns/list/campaign_list_format';
import type { CampaignListSummary } from '@/domains/campaigns/list/campaign_list_summary';
import { CAMPAIGN_LIST_FILTER_TOTALS_MAX } from '@/domains/campaigns/list/campaign_list_limits';
import {
  adminMetricPositiveClass,
  adminKpiAccentValueClass,
} from '@/lib/admin_metric_tone';
import { cn } from '@/lib/utils';
import { SummaryBand, SummaryBandDivider } from '@/shell/ui_bands';

export type CampaignListSummaryBoxProps = {
  className?: string;
  filterTotalsCapped?: boolean;
  filteredTotal?: number;
  metricsStale?: boolean;
  summary: CampaignListSummary;
};

export function CampaignListSummaryBox({
  className,
  filterTotalsCapped = false,
  filteredTotal = 0,
  metricsStale = false,
  summary,
}: CampaignListSummaryBoxProps) {
  const scopeLabel =
    summary.scope === 'selection'
      ? `${summary.rowCount} selected`
      : summary.scope === 'filter'
        ? `${summary.rowCount} filtered`
        : 'page';

  const clicks = formatTableCount(summary.clicks).text;
  const leads = formatTableCount(summary.conversions).text;
  const profit = formatTableMoneyFromMicro(summary.profitMicro).text;

  return (
    <SummaryBand className={className}>
      <p className="m-0 shrink-0 whitespace-nowrap text-xs leading-none text-muted-foreground">
        <span>{scopeLabel}: </span>
        <span className={cn('font-bold', adminKpiAccentValueClass[1])}>{clicks}</span>
        <span> clicks, </span>
        <span className={cn('font-bold', adminKpiAccentValueClass[3])}>{leads}</span>
        <span> leads, </span>
        <span className={cn('font-bold', adminMetricPositiveClass)}>{profit}</span>
        <span> profit</span>
      </p>
      {summary.marginBreachCount > 0 ? (
        <>
          <SummaryBandDivider />
          <span className="shrink-0 text-xs font-semibold text-destructive">
            Margin breach: {summary.marginBreachCount}
          </span>
        </>
      ) : null}
      {summary.staleCount > 0 ? (
        <>
          <SummaryBandDivider />
          <span className="shrink-0 whitespace-nowrap text-[11px] leading-none text-muted-foreground">
            {summary.scope === 'filter'
              ? 'Filtered totals may be stale'
              : `Stale stats: ${summary.staleCount}`}
          </span>
        </>
      ) : null}
      {filterTotalsCapped ? (
        <>
          <SummaryBandDivider />
          <span className="shrink-0 whitespace-nowrap text-[11px] leading-none text-muted-foreground">
            Filter totals unavailable above {CAMPAIGN_LIST_FILTER_TOTALS_MAX.toLocaleString()}{' '}
            campaigns ({filteredTotal.toLocaleString()} matched)
          </span>
        </>
      ) : null}
      {metricsStale && !filterTotalsCapped && summary.staleCount === 0 ? (
        <>
          <SummaryBandDivider />
          <span className="shrink-0 text-[11px] italic text-muted-foreground">
            Page metrics may be stale
          </span>
        </>
      ) : null}
    </SummaryBand>
  );
}
