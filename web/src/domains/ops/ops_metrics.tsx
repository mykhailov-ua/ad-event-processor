import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { FilterField } from '@/shell/filter_panel';
import { adminKit } from '@/lib/admin_kit';
import { EmptyState } from '@/shell/empty_state';
import type { DashboardMetrics, DashboardSummary } from '@/api/types';
import { cn } from '@/lib/utils';
import { OpsMetricsLiveSummary } from '@/domains/ops/ops_metrics_live_summary';
import {
  OpsMetricsPointsTable,
  OpsMetricsSnapshotMeta,
} from '@/domains/ops/ops_metrics_points_table';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';

export type OpsMetricsProps = {
  metrics: DashboardMetrics | undefined;
  liveSummary: DashboardSummary | undefined;
  liveEnabled: boolean;
  draftRange: string;
  fetching: boolean;
  error: Error | undefined;
  streamError: Error | undefined;
  hasSnapshot: boolean;
  onDraftRangeChange: (value: string) => void;
  onLoad: () => void;
  onLiveEnabledChange: (enabled: boolean) => void;
};

export function OpsMetrics({
  metrics,
  liveSummary,
  liveEnabled,
  draftRange,
  fetching,
  error,
  streamError,
  hasSnapshot,
  onDraftRangeChange,
  onLoad,
  onLiveEnabledChange,
}: OpsMetricsProps) {
  const points = metrics?.points ?? [];

  return (
    <OpsPageWithLoad
      alerts={streamError ? opsPanelError(streamError, 'Live stream failed') : null}
      blockingErrorTitle="Could not load dashboard metrics"
      fetchState={{ fetching, error, hasSnapshot }}
      refreshErrorTitle="Dashboard metrics refresh failed"
      title="Dashboard metrics"
      filters={
        <FilterField htmlFor="metrics-range" label="Range">
          <Input
            id="metrics-range"
            placeholder="1h"
            value={draftRange}
            onChange={(event) => onDraftRangeChange(event.target.value)}
          />
        </FilterField>
      }
      actions={
        <>
          <OpsActionGroup label="Metrics">
            <Button disabled={fetching} loading={fetching} type="button" onClick={onLoad}>
              Load metrics
            </Button>
          </OpsActionGroup>
          <OpsActionGroup label="Live stream">
            <Button
              className={cn(liveEnabled && 'border-primary bg-primary text-primary-foreground')}
              type="button"
              variant="secondary"
              onClick={() => onLiveEnabledChange(!liveEnabled)}
            >
              {liveEnabled ? 'Live on' : 'Live'}
            </Button>
          </OpsActionGroup>
        </>
      }
    >
      {liveSummary ? <OpsMetricsLiveSummary summary={liveSummary} /> : null}

      <OpsMetricsSnapshotMeta draftRange={draftRange} metrics={metrics} />

      {points.length === 0 && hasSnapshot ? (
        <EmptyState
          description="Handler returned an empty points array."
          title="No metric points"
        />
      ) : null}

      <OpsMetricsPointsTable points={points} />
    </OpsPageWithLoad>
  );
}
