import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import { getCampaignMargin, getCampaignStats } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignMargin, CampaignStats } from '@/api/types';
import { ErrorBlock } from '@/shell/error_block';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { CampaignStatusBadge } from '@/domains/campaigns/list/campaign_status_badge';
import {
  BudgetUsedSummary,
  campaignForecastHref,
  campaignFraudHref,
  HourlyTrendChart,
  MetricRow,
  MetricTile,
  type CampaignWithMoneyDisplay,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatTableMoneyFromMicro } from '@/domains/campaigns/list/campaign_list_format';
import { displayCount, displayMoneyDecimal, displayTimestamp } from '@/lib/display';

export type CampaignOverviewSheetProps = {
  campaign: CampaignWithMoneyDisplay | null;
  customerName: string;
  onOpenChange: (open: boolean) => void;
  open: boolean;
};

function StatsUnavailable({ message }: { message: string }) {
  return <p className="text-sm text-muted-foreground">{message}</p>;
}

export function CampaignOverviewSheet({
  campaign,
  customerName,
  onOpenChange,
  open,
}: CampaignOverviewSheetProps) {
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [margin, setMargin] = useState<CampaignMargin | undefined>();
  const [statsError, setStatsError] = useState<Error | undefined>();
  const [marginError, setMarginError] = useState<Error | undefined>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || !campaign) {
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setStatsError(undefined);
    setMarginError(undefined);

    void Promise.allSettled([
      getCampaignStats(campaign.id, {}, controller.signal),
      getCampaignMargin(campaign.id, controller.signal),
    ])
      .then(([statsResult, marginResult]) => {
        if (controller.signal.aborted) {
          return;
        }

        if (statsResult.status === 'fulfilled') {
          setStats(statsResult.value);
        } else {
          const reason = statsResult.reason;
          setStats(undefined);
          setStatsError(reason instanceof Error ? reason : new Error(String(reason)));
        }

        if (marginResult.status === 'fulfilled') {
          setMargin(marginResult.value);
        } else {
          const reason = marginResult.reason;
          setMargin(undefined);
          if (!(reason instanceof ApiError && reason.status === 501)) {
            setMarginError(reason instanceof Error ? reason : new Error(String(reason)));
          }
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => {
      controller.abort();
    };
  }, [campaign, open]);

  const handleOpenChange = (next: boolean) => {
    onOpenChange(next);
    if (!next) {
      setStats(undefined);
      setMargin(undefined);
      setStatsError(undefined);
      setMarginError(undefined);
      setLoading(false);
    }
  };

  if (!campaign) {
    return null;
  }

  const dailyBudget = displayMoneyDecimal(campaign.daily_budget);
  const pacingMode = campaign.pacing_mode?.trim() || '-';
  const timezone = campaign.timezone?.trim() || '-';

  return (
    <Sheet onOpenChange={handleOpenChange} open={open}>
      <SheetContent className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="pr-8">{campaign.name}</SheetTitle>
          <SheetDescription>
            <Link
              className="text-primary hover:underline"
              to={`/customers/${campaign.customer_id}`}
            >
              {customerName}
            </Link>
          </SheetDescription>
        </SheetHeader>

        <div className="mt-6 grid gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <CampaignStatusBadge campaign={campaign} />
            <span className="text-xs text-muted-foreground">
              Updated {displayTimestamp(campaign.updated_at, campaign.updated_at_display)}
            </span>
          </div>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium">Budget</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
              <BudgetUsedSummary campaign={campaign} className="max-w-none" />
              <MetricRow
                label="Budget limit"
                value={
                  displayMoneyDecimal(campaign.budget_limit, campaign.budget_limit_display) || '-'
                }
              />
              <MetricRow
                label="Current spend"
                value={
                  displayMoneyDecimal(campaign.current_spend, campaign.current_spend_display) ||
                  '-'
                }
              />
              <MetricRow label="Daily budget" value={dailyBudget || '-'} />
              <MetricRow label="Pacing" value={pacingMode} />
              <MetricRow label="Timezone" value={timezone} />
              {campaign.margin_breach ? (
                <p className="text-xs font-medium text-destructive">Margin breach flagged</p>
              ) : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium">Delivery</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
              {loading ? <p className="text-sm text-muted-foreground">Loading stats...</p> : null}

              {statsError ? (
                statsError instanceof ApiError && statsError.status === 501 ? (
                  <StatsUnavailable message="Delivery stats are not available in this environment." />
                ) : (
                  <ErrorBlock title="Could not load stats" message={statsError.message} />
                )
              ) : null}

              {stats || loading ? (
                <>
                  <div className="grid grid-cols-3 gap-2">
                    <MetricTile
                      label="Impressions"
                      value={stats ? displayCount(stats.metrics?.impressions) : '-'}
                    />
                    <MetricTile
                      label="Clicks"
                      value={stats ? displayCount(stats.metrics?.clicks) : '-'}
                    />
                    <MetricTile
                      label="Conversions"
                      value={stats ? displayCount(stats.metrics?.conversions) : '-'}
                    />
                  </div>
                  {stats?.stale ? (
                    <p className="text-xs text-muted-foreground">
                      Stats may be stale ({stats.source}).
                    </p>
                  ) : null}
                  <HourlyTrendChart buckets={stats?.hourly ?? []} height={88} loading={loading} />
                </>
              ) : null}

              {margin ? (
                <div className="grid gap-2 border-t border-border pt-3 text-sm">
                  <p className="font-medium">Margin</p>
                  <MetricRow
                    label="Operator margin"
                    value={formatTableMoneyFromMicro(margin.operator_margin_micro).text}
                  />
                  <MetricRow
                    label="Advertiser spend"
                    value={formatTableMoneyFromMicro(margin.advertiser_spend_micro).text}
                  />
                  <MetricRow
                    label="Margin breach"
                    value={margin.margin_breach ? 'Yes' : 'No'}
                  />
                </div>
              ) : null}

              {marginError ? (
                <ErrorBlock title="Could not load margin" message={marginError.message} />
              ) : null}
            </CardContent>
          </Card>

          <div className="flex flex-wrap gap-2">
            <Button asChild>
              <Link to={`/campaigns/${campaign.id}/edit`}>Edit</Link>
            </Button>
            <Button asChild>
              <Link to={`/dashboards/campaign/${campaign.id}`}>Report</Link>
            </Button>
            <Button asChild variant="secondary">
              <Link to={campaignFraudHref(campaign.id, campaign.customer_id)}>
                Fraud explain
              </Link>
            </Button>
            <Button asChild variant="secondary">
              <Link to={campaignForecastHref(campaign.customer_id)}>Forecast</Link>
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
