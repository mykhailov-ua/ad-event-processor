import { Badge } from '@/components/ui/badge';
import {
  exportJobCanCancel,
  exportJobCanDownload,
  exportJobPhase,
  formatExportJobRowSummary,
  normalizeExportJobStatus,
} from '@/domains/exports/export_hub_job_status';
import { exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import { useExportJobElapsed } from '@/domains/exports/use_export_job_elapsed';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { Button } from '@/components/ui/button';
import { ErrorBlock } from '@/shell/error_block';
import { cn } from '@/lib/utils';

export type ExportHubJobLifecycleProps = {
  jobId: string;
  status: string | undefined;
  errorMessage?: string;
  bytes?: number;
  rowLimit?: number;
  startedAtMs?: number;
  exportBusy: boolean;
  autoPolling: boolean;
  cancelling: boolean;
  downloading: boolean;
  polling: boolean;
  canCancelKind: boolean;
  onCancelJob: () => void;
  onDownloadJob: () => void;
  onPollJob: () => void;
};

export function ExportJobStatusBadge({
  status,
  elapsed,
}: {
  status: string | undefined;
  elapsed: string;
}) {
  const normalized = normalizeExportJobStatus(status);
  if (!normalized) {
    return null;
  }

  const phase = exportJobPhase(status);

  if (phase === 'pending') {
    return (
      <Badge variant="outline">
        Running{elapsed ? ` · ${elapsed}` : '...'}
      </Badge>
    );
  }

  if (phase === 'completed') {
    return <Badge variant="outline">Completed</Badge>;
  }

  if (phase === 'failed') {
    return <Badge variant="outline">Failed</Badge>;
  }

  if (phase === 'cancelled') {
    return <Badge variant="outline">Cancelled</Badge>;
  }

  return <Badge variant="outline">{normalized}</Badge>;
}

export function ExportHubJobLifecycle({
  jobId,
  status,
  errorMessage,
  bytes,
  rowLimit,
  startedAtMs,
  exportBusy,
  autoPolling,
  cancelling,
  downloading,
  polling,
  canCancelKind,
  onCancelJob,
  onDownloadJob,
  onPollJob,
}: ExportHubJobLifecycleProps) {
  const phase = exportJobPhase(status);
  const elapsed = useExportJobElapsed(phase === 'pending', startedAtMs);
  const canDownload = exportJobCanDownload(status);
  const canCancel = canCancelKind && exportJobCanCancel(status);
  const sizeSummary =
    phase === 'completed' ? formatExportJobRowSummary(rowLimit, bytes) : undefined;
  const failedJobMessage = exportHubJobErrorMessage(errorMessage);
  const hasStatus = Boolean(normalizeExportJobStatus(status));
  const trimmedJobId = jobId.trim();

  return (
    <div className={cn('grid min-w-0', adminSpacing.gap.md)}>
      {hasStatus ? (
        <div className={adminSpacing.flex.buttonGroup}>
          <ExportJobStatusBadge elapsed={elapsed} status={status} />
          {autoPolling && phase === 'pending' ? (
            <span className={adminTypography.bodyMuted}>Auto-refreshing every 3s</span>
          ) : null}
          {sizeSummary ? <span className={adminTypography.bodyMuted}>{sizeSummary}</span> : null}
        </div>
      ) : null}

      {phase === 'failed' && failedJobMessage ? (
        <ErrorBlock message={failedJobMessage} title="Export failed" />
      ) : null}

      <div className={adminSpacing.flex.buttonGroup}>
        {canCancel ? (
          <Button
            disabled={exportBusy || !trimmedJobId}
            type="button"
            variant="destructive"
            onClick={onCancelJob}
          >
            {cancelling ? 'Cancelling...' : 'Cancel job'}
          </Button>
        ) : null}
        <Button
          disabled={exportBusy || !canDownload || !trimmedJobId}
          title={
            canDownload
              ? sizeSummary
                ? `Download ${sizeSummary}`
                : 'Download completed export'
              : 'Download unlocks when the job completes'
          }
          type="button"
          onClick={onDownloadJob}
        >
          {downloading ? 'Downloading...' : 'Download'}
        </Button>
        <Button
          disabled={exportBusy || !trimmedJobId}
          type="button"
          variant="outline"
          onClick={onPollJob}
        >
          {polling ? 'Polling...' : 'Refresh status'}
        </Button>
      </div>
    </div>
  );
}
