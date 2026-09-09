import { memo } from 'react';

import type { DashboardSummary } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type OpsMetricsLiveSummaryProps = {
  summary: DashboardSummary;
};

export const OpsMetricsLiveSummary = memo(function OpsMetricsLiveSummary({
  summary,
}: OpsMetricsLiveSummaryProps) {
  return (
    <p className="text-muted-foreground" >
      Live stream / outbox pending {summary.outbox_pending ?? ''} / generated{' '}
      {displayTimestamp(summary.generated_at, summary.generated_at_display)}
    </p>
  );
});
