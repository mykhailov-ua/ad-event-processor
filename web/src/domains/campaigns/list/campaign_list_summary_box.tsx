import { formatTableCount, formatTableMoneyFromMicro } from '@/domains/campaigns/list/campaign_list_format';
import type { CampaignListSummary } from '@/domains/campaigns/list/campaign_list_summary';
import { CAMPAIGN_LIST_FILTER_TOTALS_MAX } from '@/domains/campaigns/list/campaign_list_limits';
import { cn } from '@/lib/utils';

export type CampaignListSummaryBoxProps = {
  className?: string;
  filterTotalsCapped?: boolean;
  filteredTotal?: number;
  metricsStale?: boolean;
  summary: CampaignListSummary;
};

function SummaryDivider() {
  return <span aria-hidden className="h-3 w-px shrink-0 bg-border" />;
}

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
    <div
      className={cn(
        'inline-flex max-w-full flex-wrap items-center gap-2 rounded-md border border-border bg-card px-2.5 py-1 text-card-foreground',
        className,
      )}
    >
      <p className="m-0 truncate text-xs leading-[16px] text-muted-foreground">
        <span>{scopeLabel}: </span>
        <span className="font-bold text-foreground">{clicks}</span>
        <span> clicks, </span>
        <span className="font-bold text-foreground">{leads}</span>
        <span> leads, </span>
        <span className="font-bold text-foreground">{profit}</span>
        <span> profit</span>
      </p>
      {summary.marginBreachCount > 0 ? (
        <>
          <SummaryDivider />
          <span className="shrink-0 text-xs font-semibold text-destructive">
            Margin breach: {summary.marginBreachCount}
          </span>
        </>
      ) : null}
      {summary.staleCount > 0 ? (
        <>
          <SummaryDivider />
          <span className="shrink-0 text-[11px] italic text-muted-foreground">
            {summary.scope === 'filter' ? 'Filtered totals may be stale' : `Stale stats: ${summary.staleCount}`}
          </span>
        </>
      ) : null}
      {filterTotalsCapped ? (
        <>
          <SummaryDivider />
          <span className="shrink-0 text-[11px] text-muted-foreground">
            Filter totals unavailable above {CAMPAIGN_LIST_FILTER_TOTALS_MAX.toLocaleString()} campaigns (
            {filteredTotal.toLocaleString()} matched)
          </span>
        </>
      ) : null}
      {metricsStale && !filterTotalsCapped && summary.staleCount === 0 ? (
        <>
          <SummaryDivider />
          <span className="shrink-0 text-[11px] italic text-muted-foreground">Page metrics may be stale</span>
        </>
      ) : null}
    </div>
  );
}
