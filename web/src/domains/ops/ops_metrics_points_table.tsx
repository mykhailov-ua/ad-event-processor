import { memo } from 'react';

import type { DashboardMetrics } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

export type OpsMetricsPointsTableProps = {
  metrics: DashboardMetrics | undefined;
  draftRange: string;
};

export const OpsMetricsSnapshotMeta = memo(function OpsMetricsSnapshotMeta({
  metrics,
  draftRange,
}: OpsMetricsPointsTableProps) {
  if (!metrics) {
    return null;
  }

  return (
    <p className="mb-3 text-muted-foreground" >
      Range {metrics.range ?? draftRange} / bucket {metrics.bucket_sec ?? ''}s / generated{' '}
      {displayTimestamp(metrics.generated_at)}
    </p>
  );
});

export type OpsMetricsPointRow = NonNullable<DashboardMetrics['points']>[number];

export type OpsMetricsPointsBodyProps = {
  points: OpsMetricsPointRow[];
};

export const OpsMetricsPointsTable = memo(function OpsMetricsPointsTable({
  points,
}: OpsMetricsPointsBodyProps) {
  if (points.length === 0) {
    return null;
  }

  return (
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
  );
});
