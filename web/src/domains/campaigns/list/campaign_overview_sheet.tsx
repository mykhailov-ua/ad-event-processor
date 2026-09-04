import { useEffect, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { AlertTriangle } from 'lucide-react';

import { getCampaignMargin, getCampaignStats } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignMargin, CampaignStats, CampaignStatsQuery } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import {
  campaignForecastHref,
  campaignFraudHref,
  type CampaignWithMoneyDisplay,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatTableMoneyFromMicro } from '@/domains/campaigns/list/campaign_list_format';
import {
  resolveCampaignStatusKey,
} from '@/domains/campaigns/list/campaign_list_row_tone';
import { ErrorBlock } from '@/shell/error_block';
import {
  campaignBudgetUsedPercent,
  formatBudgetUsedPercent,
} from '@/lib/campaign_budget_used';
import { displayCount, displayMoneyDecimal, displayTimestamp } from '@/lib/display';
import { isUuidLike } from '@/lib/customer_label';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { cn } from '@/lib/utils';

export type CampaignOverviewSheetProps = {
  campaign: CampaignWithMoneyDisplay | null;
  customerName: string;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  statsQuery?: CampaignStatsQuery;
};

function overviewMoney(value?: string | null, display?: string | null): string {
  const raw = displayMoneyDecimal(value, display);
  if (!raw) {
    return '-';
  }
  if (/^usd\b/i.test(raw)) {
    return raw;
  }
  return `USD ${raw}`;
}

function formatDeliveryPercent(numerator: number, denominator: number): number {
  if (denominator <= 0 || numerator <= 0) {
    return 0;
  }
  return Math.round((numerator / denominator) * 100);
}

function OverviewSection({
  children,
  className,
  title,
  meta,
}: {
  children: ReactNode;
  className?: string;
  title: string;
  meta?: React.ReactNode;
}) {
  return (
    <section className={cn('campaign-overview-dialog__section', className)}>
      <div className="mb-3 flex items-center justify-between gap-2">
        <h3 className="campaign-overview-dialog__section-title">{title}</h3>
        {meta}
      </div>
      {children}
    </section>
  );
}

function OverviewRow({
  label,
  value,
  valueClassName,
}: {
  label: string;
  value: string;
  valueClassName?: string;
}) {
  return (
    <div className="campaign-overview-dialog__row">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn('tabular-nums text-foreground', valueClassName)}>{value}</span>
    </div>
  );
}

function DeliveryMetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="campaign-overview-dialog__metric-card">
      <p className="campaign-overview-dialog__metric-card-label">{label}</p>
      <p className="campaign-overview-dialog__metric-card-value">{value}</p>
    </div>
  );
}

function DeliveryRateRow({ label, percent }: { label: string; percent: number }) {
  return (
    <div className="campaign-overview-dialog__rate-row">
      <div className="flex items-center justify-between gap-2 text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground">
        <span>{label}</span>
        <span className="tabular-nums">{percent}%</span>
      </div>
      <div aria-hidden className="h-1 overflow-hidden rounded-full bg-muted">
        <div className="h-full rounded-full bg-primary" style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

function BudgetProgress({
  campaign,
}: {
  campaign: CampaignWithMoneyDisplay;
}) {
  const percent =
    typeof campaign.budget_used_pct === 'number'
      ? campaign.budget_used_pct
      : campaignBudgetUsedPercent(campaign.budget_limit, campaign.current_spend);
  const spendLabel = overviewMoney(campaign.current_spend, campaign.current_spend_display);
  const budgetLabel = overviewMoney(campaign.budget_limit, campaign.budget_limit_display);

  return (
    <div className="mb-1 grid gap-2">
      <div aria-hidden className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            'h-full rounded-full bg-primary transition-all',
            percent != null && percent >= 90 && 'bg-destructive',
          )}
          style={{ width: `${Math.max(0, Math.min(100, percent ?? 0))}%` }}
        />
      </div>
      <p className="m-0 text-[13px] leading-[18px] tabular-nums">
        <span className="text-muted-foreground">{spendLabel}</span>
        {budgetLabel !== '-' ? <span className="text-foreground"> / {budgetLabel}</span> : null}
        {percent != null ? (
          <span className="sr-only"> ({formatBudgetUsedPercent(percent)} used)</span>
        ) : null}
      </p>
    </div>
  );
}

function StatsUnavailable({ message }: { message: string }) {
  return <p className="text-[13px] leading-[18px] text-muted-foreground">{message}</p>;
}

export function CampaignOverviewSheet({
  campaign,
  customerName,
  onOpenChange,
  open,
  statsQuery,
}: CampaignOverviewSheetProps) {
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [margin, setMargin] = useState<CampaignMargin | undefined>();
  const [statsError, setStatsError] = useState<Error | undefined>();
  const [marginError, setMarginError] = useState<Error | undefined>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || !campaign || !isUuidLike(campaign.id)) {
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setStatsError(undefined);
    setMarginError(undefined);

    void Promise.allSettled([
      getCampaignStats(campaign.id, statsQuery ?? {}, controller.signal),
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
  }, [campaign, open, statsQuery?.from, statsQuery?.granularity, statsQuery?.to]);

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
  const statusKey = resolveCampaignStatusKey(campaign.status, campaign.status_tone);
  const statusLabel = formatCampaignStatusLabel(campaign.status, campaign.status_label);
  const statusBadgeClass =
    statusKey === 'ACTIVE'
      ? 'bg-emerald-500/20 text-emerald-400'
      : statusKey === 'PAUSED'
        ? 'bg-amber-500/20 text-amber-400'
        : 'bg-muted text-muted-foreground';

  const impressions = stats?.metrics?.impressions ?? 0;
  const clicks = stats?.metrics?.clicks ?? 0;
  const conversions = stats?.metrics?.conversions ?? 0;
  const hasDeliveryActivity = impressions > 0 || clicks > 0 || conversions > 0;
  const impressionRate = impressions > 0 ? 100 : 0;
  const clickRate = formatDeliveryPercent(clicks, impressions);
  const conversionRate = formatDeliveryPercent(conversions, clicks);

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent className="campaign-overview-dialog w-[calc(100%-2rem)] max-w-md p-0">
        <div className="campaign-overview-dialog__scroll">
          <header className="campaign-overview-dialog__header">
            <div className="flex items-center justify-between gap-3">
              <span
                className={cn(
                  'inline-flex items-center rounded-[5px] px-2 py-0.5 text-[11px] font-bold uppercase leading-[14px]',
                  statusBadgeClass,
                )}
              >
                {statusLabel}
              </span>
              <span className="text-[11px] leading-[14px] text-muted-foreground">
                Updated {displayTimestamp(campaign.updated_at, campaign.updated_at_display)}
              </span>
            </div>

            <div className="grid gap-1">
              <h2 className="m-0 text-lg font-bold leading-[24px] text-foreground">{campaign.name}</h2>
              <p className="m-0 font-mono text-[13px] leading-[18px] text-muted-foreground">{campaign.id}</p>
            </div>
          </header>

          <div className="grid gap-3">
            <OverviewSection title="Budget">
              <BudgetProgress campaign={campaign} />
              <div className="campaign-overview-dialog__rows">
                <OverviewRow
                  label="Budget limit"
                  value={overviewMoney(campaign.budget_limit, campaign.budget_limit_display)}
                />
                <OverviewRow
                  label="Current spend"
                  value={overviewMoney(campaign.current_spend, campaign.current_spend_display)}
                  valueClassName="text-muted-foreground"
                />
                <OverviewRow label="Daily budget" value={dailyBudget || '-'} />
                <OverviewRow label="Pacing" value={pacingMode} />
                <OverviewRow label="Timezone" value={timezone} />
              </div>
            </OverviewSection>

            <OverviewSection
              meta={
                stats?.stale ? (
                  <span className="inline-flex items-center gap-1 rounded-[5px] bg-amber-100 px-1.5 py-0.5 text-[11px] font-semibold leading-[14px] text-amber-700">
                    <AlertTriangle className="h-3 w-3 shrink-0" aria-hidden />
                    Stats may be stale
                  </span>
                ) : null
              }
              title="Delivery"
            >
              {loading ? (
                <p className="text-[13px] leading-[18px] text-muted-foreground">Loading stats...</p>
              ) : null}

              {statsError ? (
                statsError instanceof ApiError && statsError.status === 501 ? (
                  <StatsUnavailable message="Delivery stats are not available in this environment." />
                ) : (
                  <ErrorBlock title="Could not load stats" message={statsError.message} />
                )
              ) : null}

              {stats || loading ? (
                <div className="grid gap-3">
                  <div className="grid grid-cols-3 gap-2">
                    <DeliveryMetricCard
                      label="Impressions"
                      value={stats ? displayCount(stats.metrics?.impressions) : '-'}
                    />
                    <DeliveryMetricCard
                      label="Clicks"
                      value={stats ? displayCount(stats.metrics?.clicks) : '-'}
                    />
                    <DeliveryMetricCard
                      label="Conversions"
                      value={stats ? displayCount(stats.metrics?.conversions) : '-'}
                    />
                  </div>

                  <div className="campaign-overview-dialog__rates">
                    <DeliveryRateRow label="Impressions" percent={impressionRate} />
                    <DeliveryRateRow label="Clicks" percent={clickRate} />
                    <DeliveryRateRow label="Conversions" percent={conversionRate} />
                  </div>

                  {!loading && !hasDeliveryActivity ? (
                    <p className="campaign-overview-dialog__empty-banner">No delivery activity</p>
                  ) : null}
                </div>
              ) : null}
            </OverviewSection>

            <OverviewSection title="Margin">
              {loading && !margin ? (
                <p className="text-[13px] leading-[18px] text-muted-foreground">Loading margin...</p>
              ) : null}
              {margin ? (
                <div className="campaign-overview-dialog__rows">
                  <OverviewRow
                    label="Operator margin"
                    value={formatTableMoneyFromMicro(margin.operator_margin_micro).text}
                  />
                  <OverviewRow
                    label="Advertiser spend"
                    value={formatTableMoneyFromMicro(margin.advertiser_spend_micro).text}
                  />
                  <OverviewRow
                    label="Margin breach"
                    value={margin.margin_breach ? 'Yes' : 'No'}
                  />
                </div>
              ) : !loading && !marginError ? (
                <div className="campaign-overview-dialog__rows">
                  <OverviewRow label="Margin breach" value={campaign.margin_breach ? 'Yes' : 'No'} />
                </div>
              ) : null}
              {marginError ? (
                <ErrorBlock title="Could not load margin" message={marginError.message} />
              ) : null}
            </OverviewSection>
          </div>
        </div>

        <footer className="campaign-overview-dialog__footer">
          <Button asChild className="campaign-overview-dialog__btn-outline" variant="outline">
            <Link to={campaignForecastHref(campaign.customer_id)}>Forecast</Link>
          </Button>
          <Button asChild className="campaign-overview-dialog__btn-outline" variant="outline">
            <Link to={campaignFraudHref(campaign.id, campaign.customer_id)}>Fraud explain</Link>
          </Button>
          <Button asChild className="campaign-overview-dialog__btn-primary">
            <Link to={`/dashboards/campaign/${campaign.id}`}>Report</Link>
          </Button>
          <Button asChild className="campaign-overview-dialog__btn-edit" variant="outline">
            <Link to={`/campaigns/${campaign.id}/edit`}>Edit</Link>
          </Button>
        </footer>
      </DialogContent>
    </Dialog>
  );
}
