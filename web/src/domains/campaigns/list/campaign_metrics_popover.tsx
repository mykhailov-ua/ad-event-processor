import { useState, type ReactNode } from 'react';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignStats, CampaignStatsQuery } from '@/api/types';
import { ErrorBlock } from '@/shell/error_block';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  BudgetUsedSummary,
  HourlyTrendChart,
  MetricTile,
  MetricsSection,
  type CampaignWithMoneyDisplay,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { campaignMetricsPopoverPanelClass } from '@/domains/campaigns/list/campaign_list_classes';
import { useCampaignMetricsPopoverLoad } from '@/domains/campaigns/list/use_campaign_metrics_popover_load';
import { displayCount, displayMoneyDecimal } from '@/lib/display';

function MetricsPopoverBody({
  campaign,
  error,
  loading,
  onOpenOverview,
  onRefresh,
  refreshing,
  stats,
}: {
  campaign: CampaignWithMoneyDisplay;
  error: Error | undefined;
  loading: boolean;
  onOpenOverview?: (campaign: CampaignWithMoneyDisplay) => void;
  onRefresh: () => void;
  refreshing: boolean;
  stats: CampaignStats | undefined;
}) {
  const dailyBudget = displayMoneyDecimal(campaign.daily_budget);
  const pacingMode = campaign.pacing_mode?.trim() || '-';
  const timezone = campaign.timezone?.trim() || '-';
  const budgetLimit =
    displayMoneyDecimal(campaign.budget_limit, campaign.budget_limit_display) || '-';
  const currentSpend =
    displayMoneyDecimal(campaign.current_spend, campaign.current_spend_display) || '-';

  return (
    <div className="grid">
      <header className="grid gap-3 border-b border-border p-4">
        <div className="grid gap-1">
          <p className="whitespace-nowrap text-sm tabular-nums leading-snug">{campaign.name}</p>
          <p className="text-xs text-muted-foreground">Campaign metrics</p>
        </div>
        <BudgetUsedSummary campaign={campaign} className="max-w-none" />
        {campaign.margin_breach ? (
          <p className="text-xs font-medium text-destructive">Margin breach flagged</p>
        ) : null}
      </header>

      <div className="grid gap-4 p-4">
        <MetricsSection title="Budget">
          <div className="grid grid-cols-2 gap-2">
            <MetricTile label="Budget limit" value={budgetLimit} />
            <MetricTile label="Spend" value={currentSpend} />
            <MetricTile label="Daily cap" value={dailyBudget || '-'} />
            <MetricTile label="Remaining" value={formatRemaining(campaign)} />
          </div>
          <div className="flex flex-wrap gap-1.5">
            <Badge className="font-normal" variant="outline">
              {pacingMode}
            </Badge>
            <Badge className="font-normal" variant="outline">
              {timezone}
            </Badge>
          </div>
        </MetricsSection>

        <MetricsSection
          meta={
            stats?.stale ? (
              <span className="text-ui-caption text-muted-foreground">Stale ({stats.source})</span>
            ) : null
          }
          title="Delivery"
        >
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading delivery stats...</p>
          ) : null}

          {error ? (
            error instanceof ApiError && error.status === 501 ? (
              <p className="text-sm text-muted-foreground">
                Delivery stats are not available in this environment.
              </p>
            ) : (
              <ErrorBlock title="Could not load stats" message={error.message} />
            )
          ) : null}

          <div className="grid grid-cols-3 gap-2">
            <MetricTile
              label="Impressions"
              value={stats ? displayCount(stats.metrics?.impressions) : loading ? '...' : '-'}
            />
            <MetricTile
              label="Clicks"
              value={stats ? displayCount(stats.metrics?.clicks) : loading ? '...' : '-'}
            />
            <MetricTile
              label="Conversions"
              value={stats ? displayCount(stats.metrics?.conversions) : loading ? '...' : '-'}
            />
          </div>
          <HourlyTrendChart buckets={stats?.hourly ?? []} loading={loading} />
          <Button
            disabled={loading || refreshing}
            type="button"
            variant="secondary"
            onClick={onRefresh}
          >
            Refresh stats
          </Button>
        </MetricsSection>
      </div>

      {onOpenOverview ? (
        <footer className="border-t border-border p-4">
          <Button
            className="w-full"
            onClick={() => onOpenOverview(campaign)}
            type="button"
            variant="secondary"
          >
            Open overview
          </Button>
        </footer>
      ) : null}
    </div>
  );
}

function formatRemaining(campaign: CampaignWithMoneyDisplay): string {
  const budget = Number.parseFloat(campaign.budget_limit ?? '');
  const spend = Number.parseFloat(campaign.current_spend ?? '');
  if (!Number.isFinite(budget) || !Number.isFinite(spend)) {
    return '-';
  }
  const remaining = Math.max(0, budget - spend);
  return displayMoneyDecimal(remaining.toFixed(2)) || '-';
}

export function CampaignMetricsPopover({
  campaign,
  listMetrics,
  onOpenOverview,
  statsCacheRevision,
  statsQuery,
  triggerContent,
}: {
  campaign: CampaignWithMoneyDisplay;
  listMetrics?: CampaignListMetrics;
  onOpenOverview?: (campaign: CampaignWithMoneyDisplay) => void;
  statsCacheRevision: string;
  statsQuery?: CampaignStatsQuery;
  triggerContent?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const { stats, error, loading, onRefresh } = useCampaignMetricsPopoverLoad({
    open,
    campaignId: campaign.id,
    listMetrics,
    statsCacheRevision,
    statsQuery,
  });

  const handleOpenOverview = (selected: CampaignWithMoneyDisplay) => {
    setOpen(false);
    onOpenOverview?.(selected);
  };

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <PopoverTrigger asChild>
        <Button
          aria-label={`View metrics for ${campaign.name}`}
          className="block h-auto w-full min-w-0 justify-start border-0 bg-transparent p-0 text-left font-normal shadow-none hover:bg-muted/50"
          type="button"
          variant="ghost"
        >
          {triggerContent ?? <BudgetUsedSummary campaign={campaign} />}
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        collisionPadding={16}
        onOpenAutoFocus={(event) => event.preventDefault()}
        panelClassName={campaignMetricsPopoverPanelClass}
        side="bottom"
        sideOffset={8}
        sticky="partial"
      >
        <MetricsPopoverBody
          campaign={campaign}
          error={error}
          loading={loading}
          refreshing={loading}
          onOpenOverview={onOpenOverview ? handleOpenOverview : undefined}
          onRefresh={onRefresh}
          stats={stats}
        />
      </PopoverContent>
    </Popover>
  );
}
