import { useState, type ReactNode } from 'react';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignStats, CampaignStatsQuery } from '@/api/types';
import { panelError } from '@/shell/panel_error';
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
    <div >
      <header >
        <div >
          <p >{campaign.name}</p>
          <p >Campaign metrics</p>
        </div>
        <BudgetUsedSummary campaign={campaign} />
        {campaign.margin_breach ? (
          <p >Margin breach flagged</p>
        ) : null}
      </header>

      <div >
        <MetricsSection title="Budget">
          <div >
            <MetricTile label="Budget limit" value={budgetLimit} />
            <MetricTile label="Spend" value={currentSpend} />
            <MetricTile label="Daily cap" value={dailyBudget || '-'} />
            <MetricTile label="Remaining" value={formatRemaining(campaign)} />
          </div>
          <div >
            <Badge  variant="outline">
              {pacingMode}
            </Badge>
            <Badge  variant="outline">
              {timezone}
            </Badge>
          </div>
        </MetricsSection>

        <MetricsSection
          meta={
            stats?.stale ? (
              <span >Stale ({stats.source})</span>
            ) : null
          }
          title="Delivery"
        >
          {loading ? (
            <p >Loading delivery stats...</p>
          ) : null}

          {error ? (
            error instanceof ApiError && error.status === 501 ? (
              <p >
                Delivery stats are not available in this environment.
              </p>
            ) : (
              panelError(error, 'Could not load stats')
            )
          ) : null}

          <div >
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
        <footer >
          <Button
           
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
  const remaining = budget - spend;
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
