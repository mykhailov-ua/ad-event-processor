import { useEffect, useState } from 'react';

import { getCampaignStats } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignStats } from '@/api/types';
import { ErrorBlock } from '@/components/system/error_block';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  BudgetUsedSummary,
  HourlyTrendChart,
  MetricTile,
  MetricsSection,
  type CampaignWithMoneyDisplay,
} from '@/domains/campaigns/campaign_metrics_shared';
import { displayCount, displayMoneyDecimal } from '@/lib/display';

function MetricsPopoverBody({
  campaign,
  error,
  loading,
  onOpenOverview,
  stats,
}: {
  campaign: CampaignWithMoneyDisplay;
  error: Error | undefined;
  loading: boolean;
  onOpenOverview?: (campaign: CampaignWithMoneyDisplay) => void;
  stats: CampaignStats | undefined;
}) {
  const dailyBudget = displayMoneyDecimal(campaign.daily_budget);
  const pacingMode = campaign.pacing_mode?.trim() || '—';
  const timezone = campaign.timezone?.trim() || '—';
  const budgetLimit =
    displayMoneyDecimal(campaign.budget_limit, campaign.budget_limit_display) || '—';
  const currentSpend =
    displayMoneyDecimal(campaign.current_spend, campaign.current_spend_display) || '—';

  return (
    <div className="divide-y divide-border">
      <header className="grid gap-3 p-4">
        <div className="grid gap-1">
          <p className="line-clamp-2 text-sm font-medium leading-snug">{campaign.name}</p>
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
            <MetricTile label="Daily cap" value={dailyBudget || '—'} />
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
              <span className="text-[11px] text-muted-foreground">Stale ({stats.source})</span>
            ) : null
          }
          title="Delivery"
        >
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading delivery stats…</p>
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
              value={stats ? displayCount(stats.metrics?.impressions) : loading ? '…' : '—'}
            />
            <MetricTile
              label="Clicks"
              value={stats ? displayCount(stats.metrics?.clicks) : loading ? '…' : '—'}
            />
            <MetricTile
              label="Conversions"
              value={stats ? displayCount(stats.metrics?.conversions) : loading ? '…' : '—'}
            />
          </div>
          <HourlyTrendChart buckets={stats?.hourly ?? []} loading={loading} />
        </MetricsSection>
      </div>

      {onOpenOverview ? (
        <footer className="p-4 pt-0">
          <Button
            className="w-full"
            onClick={() => onOpenOverview(campaign)}
            size="sm"
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
    return '—';
  }
  const remaining = Math.max(0, budget - spend);
  return displayMoneyDecimal(remaining.toFixed(2)) || '—';
}

export function CampaignMetricsPopover({
  campaign,
  onOpenOverview,
}: {
  campaign: CampaignWithMoneyDisplay;
  onOpenOverview?: (campaign: CampaignWithMoneyDisplay) => void;
}) {
  const [open, setOpen] = useState(false);
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [error, setError] = useState<Error | undefined>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setError(undefined);

    void getCampaignStats(campaign.id, {}, controller.signal)
      .then((next) => {
        setStats(next);
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) {
          return;
        }
        setError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => {
      controller.abort();
    };
  }, [campaign.id, open]);

  const handleOpenChange = (next: boolean) => {
    setOpen(next);
    if (!next) {
      setStats(undefined);
      setError(undefined);
      setLoading(false);
    }
  };

  const handleOpenOverview = (selected: CampaignWithMoneyDisplay) => {
    handleOpenChange(false);
    onOpenOverview?.(selected);
  };

  return (
    <Popover onOpenChange={handleOpenChange} open={open}>
      <PopoverTrigger asChild>
        <button
          aria-label={`View metrics for ${campaign.name}`}
          className="block w-full min-w-0 rounded-md text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          type="button"
        >
          <BudgetUsedSummary campaign={campaign} />
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="p-0 [&_.ui-shell]:!w-[22rem] [&_.ui-shell]:!min-w-[22rem]"
        collisionPadding={16}
        onOpenAutoFocus={(event) => event.preventDefault()}
        side="bottom"
        sideOffset={8}
        sticky="partial"
      >
        <MetricsPopoverBody
          campaign={campaign}
          error={error}
          loading={loading}
          onOpenOverview={onOpenOverview ? handleOpenOverview : undefined}
          stats={stats}
        />
      </PopoverContent>
    </Popover>
  );
}
