import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { DashboardMetrics, DashboardSummary } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { cn } from '@/lib/utils';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

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
      {liveSummary ? (
        <p className="text-muted-foreground">
          Live stream  /  outbox pending {liveSummary.outbox_pending ?? ''}  /  generated{' '}
          {displayTimestamp(liveSummary.generated_at, liveSummary.generated_at_display)}
        </p>
      ) : null}

      {metrics ? (
        <p className="text-muted-foreground">
          Range {metrics.range ?? draftRange}  /  bucket {metrics.bucket_sec ?? ''}s  /  generated{' '}
          {displayTimestamp(metrics.generated_at)}
        </p>
      ) : null}

      {points.length === 0 && hasSnapshot ? (
        <EmptyState description="Handler returned an empty points array." title="No metric points" />
      ) : null}

      {points.length > 0 ? (
        <OpsTable
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>Timestamp</OpsTableHead>
              <OpsTableHead numeric>Value</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {points.map((point, index) => (
            <OpsTableRow key={`${point.ts ?? 'point'}-${index}`}>
              <OpsTableCell>{displayTimestamp(point.ts)}</OpsTableCell>
              <OpsTableCell numeric>{point.value ?? ''}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      ) : null}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
