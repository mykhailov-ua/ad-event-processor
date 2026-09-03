import type { ReactNode } from 'react';

import type { Campaign, CampaignStats } from '@/api/types';
import type { CampaignStatusTone } from '@/domains/campaigns/list/campaign_list_row_tone';
import {
  campaignBudgetUsedPercent,
  formatBudgetUsedPercent,
} from '@/lib/campaign_budget_used';
import { displayMoneyDecimal } from '@/lib/display';
import { cn } from '@/lib/utils';

export type CampaignWithMoneyDisplay = Campaign & {
  budget_limit_display?: string;
  current_spend_display?: string;
  status_label?: string;
  status_tone?: CampaignStatusTone;
};

export function campaignForecastHref(customerId: string): string {
  const search = new URLSearchParams();
  search.set('customer_id', customerId);
  return `/forecast/campaign?${search.toString()}`;
}

export function campaignFraudHref(campaignId: string, customerId: string): string {
  const search = new URLSearchParams();
  search.set('campaign_id', campaignId);
  search.set('customer_id', customerId);
  return `/fraud/decisions?${search.toString()}`;
}

export function MetricRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="tabular-nums text-foreground">{value}</span>
    </div>
  );
}

export function MetricTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl bg-muted/30 px-3 py-2">
      <p className="text-[11px] text-muted-foreground">{label}</p>
      <p className="truncate text-sm font-medium tabular-nums">{value}</p>
    </div>
  );
}

export function MetricsSection({
  children,
  meta,
  title,
}: {
  children: ReactNode;
  meta?: ReactNode;
  title: string;
}) {
  return (
    <section className="grid gap-3">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-[11px] font-medium tracking-wide text-muted-foreground">
          {title}
        </h3>
        {meta}
      </div>
      {children}
    </section>
  );
}

export function BudgetUsedSummary({
  campaign,
  className,
  showBar = true,
}: {
  campaign: CampaignWithMoneyDisplay;
  className?: string;
  showBar?: boolean;
}) {
  const percent = campaignBudgetUsedPercent(campaign.budget_limit, campaign.current_spend);
  const spendLabel = displayMoneyDecimal(campaign.current_spend, campaign.current_spend_display);
  const budgetLabel = displayMoneyDecimal(campaign.budget_limit, campaign.budget_limit_display);
  const moneySummary =
    spendLabel && budgetLabel
      ? `${spendLabel} / ${budgetLabel}`
      : spendLabel || budgetLabel || undefined;

  if (percent == null && !moneySummary) {
    return <span className="text-muted-foreground">-</span>;
  }

  return (
    <div className={cn('grid min-w-[6.5rem] max-w-[10rem] gap-1', className)}>
      <span className="tabular-nums text-sm font-medium">
        {percent != null ? formatBudgetUsedPercent(percent) : '-'}
      </span>
      {moneySummary ? (
        <p className="truncate text-xs tabular-nums text-muted-foreground">{moneySummary}</p>
      ) : null}
      {showBar && percent != null ? (
        <div aria-hidden className="h-1.5 overflow-hidden rounded-full bg-muted">
          <div
            className={cn(
              'h-full rounded-full bg-primary transition-all',
              percent >= 90 && 'bg-destructive',
            )}
            style={{ width: `${percent}%` }}
          />
        </div>
      ) : null}
    </div>
  );
}

type HourlyBucket = NonNullable<CampaignStats['hourly']>[number];

function formatHourLabel(hour?: string): string | undefined {
  if (!hour?.trim()) {
    return undefined;
  }
  const parsed = new Date(hour);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }
  return parsed.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
}

function buildLinePath(values: number[], width: number, height: number, padding: number): string {
  if (values.length === 0) {
    return '';
  }

  const max = Math.max(...values, 1);
  const innerHeight = height - padding * 2;
  const step = values.length > 1 ? width / (values.length - 1) : 0;

  return values
    .map((value, index) => {
      const x = values.length > 1 ? index * step : width / 2;
      const y = padding + innerHeight - (value / max) * innerHeight;
      return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
    })
    .join(' ');
}

function buildAreaPath(values: number[], width: number, height: number, padding: number): string {
  if (values.length === 0) {
    return '';
  }

  const line = buildLinePath(values, width, height, padding);
  const step = values.length > 1 ? width / (values.length - 1) : 0;
  const lastX = values.length > 1 ? (values.length - 1) * step : width / 2;
  const baseline = height - padding;

  return `${line} L ${lastX} ${baseline} L 0 ${baseline} Z`;
}

export function HourlyTrendChart({
  buckets,
  height = 72,
  loading = false,
}: {
  buckets: HourlyBucket[];
  height?: number;
  loading?: boolean;
}) {
  const windowBuckets = buckets.slice(-24);
  const clicks = windowBuckets.map((bucket) => bucket.clicks ?? 0);
  const impressions = windowBuckets.map((bucket) => bucket.impressions ?? 0);
  const hasActivity = clicks.some((value) => value > 0) || impressions.some((value) => value > 0);
  const width = 288;
  const padding = 6;
  const gridLines = [0.25, 0.5, 0.75];

  const firstLabel = formatHourLabel(windowBuckets[0]?.hour);
  const lastLabel = formatHourLabel(windowBuckets[windowBuckets.length - 1]?.hour);

  if (loading) {
    return (
      <div
        className="rounded-xl bg-muted/25"
        style={{ height: height + 28 }}
      >
        <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
          Loading chart...
        </div>
      </div>
    );
  }

  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span>Last 24 hours</span>
        <div className="flex items-center gap-3">
          <span className="inline-flex items-center gap-1">
            <span aria-hidden className="h-0.5 w-3 rounded-full bg-primary" />
            Clicks
          </span>
          <span className="inline-flex items-center gap-1">
            <span aria-hidden className="h-0.5 w-3 rounded-full bg-muted-foreground/60" />
            Impressions
          </span>
        </div>
      </div>

      <div className="relative rounded-xl bg-muted/20 px-2 pt-2">
        <svg
          aria-hidden
          className="block w-full"
          height={height}
          preserveAspectRatio="none"
          viewBox={`0 0 ${width} ${height}`}
          width="100%"
        >
          {gridLines.map((ratio) => {
            const y = padding + (height - padding * 2) * ratio;
            return (
              <line
                key={ratio}
                stroke="hsl(var(--border))"
                strokeWidth={1}
                x1={0}
                x2={width}
                y1={y}
                y2={y}
              />
            );
          })}

          {hasActivity ? (
            <>
              <path
                d={buildAreaPath(impressions, width, height, padding)}
                fill="hsl(var(--muted-foreground) / 0.12)"
              />
              <path
                d={buildAreaPath(clicks, width, height, padding)}
                fill="hsl(var(--primary) / 0.18)"
              />
              <path
                d={buildLinePath(impressions, width, height, padding)}
                fill="none"
                stroke="hsl(var(--muted-foreground) / 0.7)"
                strokeWidth={1.5}
              />
              <path
                d={buildLinePath(clicks, width, height, padding)}
                fill="none"
                stroke="#3b82f6"
                strokeWidth={1.75}
              />
            </>
          ) : (
            <line
              stroke="hsl(var(--border))"
              strokeDasharray="4 4"
              strokeWidth={1}
              x1={0}
              x2={width}
              y1={height - padding}
              y2={height - padding}
            />
          )}
        </svg>

        {!hasActivity ? (
          <p className="pb-2 text-center text-[11px] text-muted-foreground">
            No delivery activity in the last 24 hours
          </p>
        ) : null}

        {firstLabel || lastLabel ? (
          <div className="flex items-center justify-between px-1 pb-1.5 text-[10px] tabular-nums text-muted-foreground">
            <span>{firstLabel ?? ''}</span>
            <span>{lastLabel ?? ''}</span>
          </div>
        ) : null}
      </div>
    </div>
  );
}
