import { type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { AlertTriangle } from 'lucide-react';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignMargin, CampaignStatsQuery } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import {
  campaignForecastHref,
  campaignFraudHref,
  type CampaignWithMoneyDisplay,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatTableMoneyFromMicro } from '@/domains/campaigns/list/campaign_list_format';
import { resolveCampaignStatusKey } from '@/domains/campaigns/list/campaign_list_row_tone';
import { adminKpiAccentValueClass, adminStaleHintClass } from '@/lib/admin_metric_tone';
import {
  campaignOverviewDialogClass,
  campaignOverviewEmptyBannerClass,
  campaignOverviewFooterClass,
  campaignOverviewHeaderClass,
  campaignOverviewMetricCardAccentClass,
  campaignOverviewMetricCardClass,
  campaignOverviewMetricLabelClass,
  campaignOverviewMetricValueClass,
  campaignOverviewOutlineButtonClass,
  campaignOverviewPrimaryButtonClass,
  campaignOverviewRateRowClass,
  campaignOverviewRatesClass,
  campaignOverviewRowClass,
  campaignOverviewRowsClass,
  campaignOverviewScrollClass,
  campaignOverviewSectionClass,
  campaignOverviewSectionTitleClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { ErrorBlock } from '@/shell/error_block';
import { campaignBudgetUsedPercent, formatBudgetUsedPercent } from '@/lib/campaign_budget_used';
import { displayCount, displayMoneyDecimal, displayTimestamp } from '@/lib/display';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { useCampaignOverviewSheetLoad } from '@/domains/campaigns/list/use_campaign_overview_sheet_load';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type CampaignOverviewSheetProps = {
  campaign: CampaignWithMoneyDisplay | null;
  customerName: string;
  listMargin?: CampaignMargin;
  listMetrics?: CampaignListMetrics;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  statsCacheRevision: string;
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
    <section className={cn(campaignOverviewSectionClass, className)}>
      <div className="flex items-center justify-between gap-2">
        <h3 className={campaignOverviewSectionTitleClass}>{title}</h3>
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
    <div className={campaignOverviewRowClass}>
      <span className="text-muted-foreground">{label}</span>
      <span className={cn('tabular-nums text-foreground', valueClassName)}>{value}</span>
    </div>
  );
}

function DeliveryMetricCard({
  label,
  value,
  accent,
}: {
  label: string;
  value: string;
  accent: 1 | 2 | 3;
}) {
  return (
    <div className={cn(campaignOverviewMetricCardClass, campaignOverviewMetricCardAccentClass[accent])}>
      <p className={cn('m-0', campaignOverviewMetricLabelClass)}>{label}</p>
      <p className={cn('m-0', campaignOverviewMetricValueClass, adminKpiAccentValueClass[accent])}>
        {value}
      </p>
    </div>
  );
}

function DeliveryRateRow({ label, percent }: { label: string; percent: number }) {
  return (
    <div className={campaignOverviewRateRowClass}>
      <div className="flex items-center justify-between gap-2 text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground">
        <span>{label}</span>
        <span className="tabular-nums">{percent}%</span>
      </div>
      <div aria-hidden className="h-1 overflow-hidden rounded-full bg-muted">
        <div className="h-full rounded-full bg-chart-1 transition-all" style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

function BudgetProgress({ campaign }: { campaign: CampaignWithMoneyDisplay }) {
  const percent =
    typeof campaign.budget_used_pct === 'number'
      ? campaign.budget_used_pct
      : campaignBudgetUsedPercent(campaign.budget_limit, campaign.current_spend);
  const spendLabel = overviewMoney(campaign.current_spend, campaign.current_spend_display);
  const budgetLabel = overviewMoney(campaign.budget_limit, campaign.budget_limit_display);

  return (
    <div className="grid gap-2">
      <div aria-hidden className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            'h-full rounded-full bg-primary transition-all',
            percent != null && percent >= 90 && 'bg-destructive'
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
  listMargin,
  listMetrics,
  onOpenChange,
  open,
  statsCacheRevision,
  statsQuery,
}: CampaignOverviewSheetProps) {
  const { stats, margin, statsError, marginError, loading, reset } = useCampaignOverviewSheetLoad({
    open,
    campaign,
    listMargin,
    listMetrics,
    statsCacheRevision,
    statsQuery,
  });

  const handleOpenChange = (next: boolean) => {
    onOpenChange(next);
    if (!next) {
      reset();
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
      ? 'bg-admin-status-active/20 text-admin-status-active'
      : statusKey === 'PAUSED'
        ? 'bg-admin-status-paused/20 text-admin-status-paused'
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
      <DialogContent className={campaignOverviewDialogClass}>
        <div className={campaignOverviewScrollClass}>
          <header className={campaignOverviewHeaderClass}>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <span
                className={cn(
                  'inline-flex items-center px-2 py-0.5 text-[11px] font-bold uppercase leading-[14px]',
                  adminKit.controlRadius,
                  statusBadgeClass
                )}
              >
                {statusLabel}
              </span>
              <span className="min-w-0 text-[11px] leading-[14px] text-muted-foreground">
                Updated {displayTimestamp(campaign.updated_at, campaign.updated_at_display)}
              </span>
            </div>

            <div className="grid min-w-0 gap-1">
              <h2 className="m-0 whitespace-nowrap text-lg tabular-nums leading-[24px] text-foreground">
                {campaign.name}
              </h2>
              <p className="m-0 break-all font-mono text-[13px] leading-[18px] text-muted-foreground">
                {campaign.id}
              </p>
            </div>
          </header>

          <div className="grid gap-3">
            <OverviewSection title="Budget">
              <BudgetProgress campaign={campaign} />
              <div className={campaignOverviewRowsClass}>
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
                  <span className={cn(adminStaleHintClass, adminKit.controlRadius)}>
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
                      accent={1}
                      label="Impressions"
                      value={stats ? displayCount(stats.metrics?.impressions) : '-'}
                    />
                    <DeliveryMetricCard
                      accent={2}
                      label="Clicks"
                      value={stats ? displayCount(stats.metrics?.clicks) : '-'}
                    />
                    <DeliveryMetricCard
                      accent={3}
                      label="Conversions"
                      value={stats ? displayCount(stats.metrics?.conversions) : '-'}
                    />
                  </div>

                  <div className={campaignOverviewRatesClass}>
                    <DeliveryRateRow label="Impressions" percent={impressionRate} />
                    <DeliveryRateRow label="Clicks" percent={clickRate} />
                    <DeliveryRateRow label="Conversions" percent={conversionRate} />
                  </div>

                  {!loading && !hasDeliveryActivity ? (
                    <p className={campaignOverviewEmptyBannerClass}>No delivery activity</p>
                  ) : null}
                </div>
              ) : null}
            </OverviewSection>

            <OverviewSection title="Margin">
              {loading && !margin ? (
                <p className="text-[13px] leading-[18px] text-muted-foreground">
                  Loading margin...
                </p>
              ) : null}
              {margin ? (
                <div className={campaignOverviewRowsClass}>
                  <OverviewRow
                    label="Operator margin"
                    value={formatTableMoneyFromMicro(margin.operator_margin_micro).text}
                  />
                  <OverviewRow
                    label="Advertiser spend"
                    value={formatTableMoneyFromMicro(margin.advertiser_spend_micro).text}
                  />
                  <OverviewRow label="Margin breach" value={margin.margin_breach ? 'Yes' : 'No'} />
                </div>
              ) : !loading && !marginError ? (
                <div className={campaignOverviewRowsClass}>
                  <OverviewRow
                    label="Margin breach"
                    value={campaign.margin_breach ? 'Yes' : 'No'}
                  />
                </div>
              ) : null}
              {marginError ? (
                <ErrorBlock title="Could not load margin" message={marginError.message} />
              ) : null}
            </OverviewSection>
          </div>
        </div>

        <footer className={campaignOverviewFooterClass}>
          <Button asChild className={campaignOverviewOutlineButtonClass} variant="outline">
            <Link to={campaignForecastHref(campaign.customer_id)}>Forecast</Link>
          </Button>
          <Button asChild className={campaignOverviewOutlineButtonClass} variant="outline">
            <Link to={campaignFraudHref(campaign.id, campaign.customer_id)}>Fraud explain</Link>
          </Button>
          <Button asChild className={campaignOverviewPrimaryButtonClass}>
            <Link to={`/dashboards/campaign/${campaign.id}`}>Report</Link>
          </Button>
          <Button asChild className={campaignOverviewOutlineButtonClass} variant="outline">
            <Link to={`/campaigns/${campaign.id}/edit`}>Edit</Link>
          </Button>
        </footer>
      </DialogContent>
    </Dialog>
  );
}
