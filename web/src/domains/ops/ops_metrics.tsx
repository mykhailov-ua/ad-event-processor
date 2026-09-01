import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { DashboardMetrics, DashboardSummary } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

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
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Dashboard metrics">
        <OpsNav />
        {opsPanelError(error, 'Could not load dashboard metrics')}
      </PageChrome>
    );
  }

  const points = metrics?.points ?? [];

  return (
    <PageChrome title="Dashboard metrics">
      <OpsNav />

      <div className="grid max-w-md grid-cols-[1fr_auto_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="metrics-range">Range</Label>
          <Input
            id="metrics-range"
            value={draftRange}
            placeholder="1h"
            onChange={(event) => onDraftRangeChange(event.target.value)}
          />
        </div>
        <PrimaryActionButton disabled={fetching} loading={fetching} onClick={onLoad} type="button">
          Load metrics
        </PrimaryActionButton>
        <SecondaryActionButton
          onClick={() => onLiveEnabledChange(!liveEnabled)}
          type="button"
          variant={liveEnabled ? 'default' : 'outline'}
        >
          {liveEnabled ? 'Live on' : 'Live'}
        </SecondaryActionButton>
      </div>

      {liveSummary ? (
        <p className="text-sm text-muted-foreground">
          Live stream * outbox pending {liveSummary.outbox_pending ?? ''} * generated{' '}
          {displayTimestamp(liveSummary.generated_at, liveSummary.generated_at_display)}
        </p>
      ) : null}

      {metrics ? (
        <p className="text-sm text-muted-foreground">
          Range {metrics.range ?? draftRange} * bucket {metrics.bucket_sec ?? ''}s * generated{' '}
          {displayTimestamp(metrics.generated_at)}
        </p>
      ) : null}

      {points.length === 0 && hasSnapshot ? (
        <EmptyState title="No metric points" description="Handler returned an empty points array." />
      ) : null}

      {points.length > 0 ? (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Timestamp</TableHead>
                <TableHead className="text-right">Value</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {points.map((point, index) => (
                <TableRow key={`${point.ts ?? 'point'}-${index}`}>
                  <TableCell>{displayTimestamp(point.ts)}</TableCell>
                  <TableCell className="text-right tabular-nums">{point.value ?? ''}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
