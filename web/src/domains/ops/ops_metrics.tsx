import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { DashboardMetrics, DashboardSummary } from '@/api/types';
import { cn } from '@/lib/utils';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsMetricsLiveSummary } from '@/domains/ops/ops_metrics_live_summary';
import {
  OpsMetricsPointsTable,
  OpsMetricsSnapshotMeta,
} from '@/domains/ops/ops_metrics_points_table';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';

export type OpsMetricsProps = {
  metrics: DashboardMetrics | undefined;
  liveSummary: DashboardSummary | undefined;
  liveEnabled: boolean;
  draftRange: string;
  fetching: boolean;
  error: Error | undefined;
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
  hasSnapshot,
  onDraftRangeChange,
  onLoad,
  onLiveEnabledChange,
}: OpsMetricsProps) {
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError
        error={error}
        pageTitle="Dashboard metrics"
        title="Could not load dashboard metrics"
      />
    );
  }

  const points = metrics?.points ?? [];

  return (
    <OpsPageShell
      filters={
        <div className="grid gap-2">
          <Label htmlFor="metrics-range">Range</Label>
          <Input
            id="metrics-range"
            placeholder="1h"
            value={draftRange}
            onChange={(event) => onDraftRangeChange(event.target.value)}
          />
        </div>
      }
      title="Dashboard metrics"
      actions={
        <>
          <OpsActionGroup label="Metrics">
            <Button disabled={fetching} loading={fetching} type="button" onClick={onLoad}>
              Load metrics
            </Button>
          </OpsActionGroup>
          <OpsActionGroup label="Live stream">
            <Button
              className={cn(liveEnabled && 'bg-primary text-primary-foreground')}
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

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
