import { Link } from 'react-router-dom';

import type { CampaignStats, CampaignStatsQuery } from '@/api/types';
import {
  HourlyTrendChart,
  MetricTile,
  MetricsSection,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { buildCampaignStatsReportHref, campaignReportPath } from '@/lib/campaign_nav';
import { displayCount, displayMoneyDecimal, displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';
import { adminSpacing } from '@/lib/admin_spacing';
import { ErrorBlock } from '@/shell/error_block';
import { CampaignManualCostForm } from '@/domains/campaigns/editor/campaign_manual_cost_form';
import { BentoSection } from '@/shell/bento_card';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

type CampaignDailyRow = NonNullable<CampaignStats['daily']>[number];

export type CampaignStatsPanelProps = {
  campaignId: string;
  customerId?: string;
  stats?: CampaignStats;
  statsQuery?: CampaignStatsQuery;
  loading?: boolean;
  error?: Error;
  onReload: () => void;
};

function CampaignDailyBucketList({ rows }: { rows: CampaignDailyRow[] }) {
  const buckets = rows.slice(-14);
  if (buckets.length === 0) {
    return <p className={adminTypography.bodyMuted}>No daily buckets in range.</p>;
  }
  return (
    <ul className={`grid ${adminSpacing.gap.xs}`}>
      {buckets.map((row) => (
        <li
          key={row.day ?? `${row.impressions}-${row.clicks}`}
          className={cn(
            'grid grid-cols-[1fr_repeat(3,minmax(0,auto))] gap-3',
            adminTypography.captionPlain
          )}
        >
          <span>{displayTimestamp(row.day) || '-'}</span>
          <span>{displayCount(row.impressions)} imp</span>
          <span>{displayCount(row.clicks)} clk</span>
          <span>{displayCount(row.conversions)} conv</span>
        </li>
      ))}
    </ul>
  );
}

export function CampaignStatsPanel({
  campaignId,
  customerId,
  stats,
  statsQuery,
  loading = false,
  error,
  onReload,
}: CampaignStatsPanelProps) {
  const reportHref = buildCampaignStatsReportHref({
    campaignId,
    customerId,
    from: statsQuery?.from,
    to: statsQuery?.to,
    granularity: statsQuery?.granularity,
  });
  const exportHref = campaignReportPath(campaignId);

  return (
    <BentoSection title="Campaign stats">
      <div className={adminSpacing.flex.buttonGroup}>
        <Button disabled={loading} onClick={onReload} type="button" variant="outline">
          {loading ? 'Loading...' : stats ? 'Refresh stats' : 'Load stats'}
        </Button>
        <Button asChild type="button" variant="secondary">
          <Link to={reportHref}>Open campaign stats report</Link>
        </Button>
        <Button asChild type="button" variant="ghost">
          <Link to={exportHref}>Export campaign-stats</Link>
        </Button>
      </div>

      {error ? <ErrorBlock error={error} title="Could not load campaign stats" /> : null}

      {!stats && !loading && !error ? (
        <p className={adminTypography.bodyMuted}>
          Load stats for hourly and daily delivery in the selected range.
        </p>
      ) : null}

      {stats ? (
        <>
          <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
            <MetricTile label="Spend" value={displayMoneyDecimal(stats.current_spend) || '-'} />
            <MetricTile
              label="Impressions"
              value={displayCount(stats.metrics?.impressions) || '0'}
            />
            <MetricTile label="Clicks" value={displayCount(stats.metrics?.clicks) || '0'} />
            <MetricTile
              label="Conversions"
              value={displayCount(stats.metrics?.conversions) || '0'}
            />
          </div>

          <MetricsSection title="Hourly trend">
            <HourlyTrendChart buckets={stats.hourly ?? []} loading={loading} />
          </MetricsSection>

          <MetricsSection title="Daily buckets">
            <CampaignDailyBucketList rows={stats.daily ?? []} />
          </MetricsSection>

          <MetricsSection title="Manual cost entry">
            <CampaignManualCostForm campaignId={campaignId} onSaved={onReload} />
          </MetricsSection>
        </>
      ) : null}
    </BentoSection>
  );
}
