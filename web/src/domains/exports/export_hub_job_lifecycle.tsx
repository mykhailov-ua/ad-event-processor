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
import { Button } from '@/components/ui/button';
import { ErrorBlock } from '@/shell/error_block';

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
  canCancelKind: boolean;
  onCancelJob: () => void;
  onDownloadJob: () => void;
};

export function ExportJobStatusBadge({
  status,
  elapsed,
}: {
  status: string | undefined;
  elapsed: string;
}) {
  const phase = exportJobPhase(status);
  const label = normalizeExportJobStatus(status) || 'unknown';

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

  return <Badge variant="outline">{label}</Badge>;
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
  canCancelKind,
  onCancelJob,
  onDownloadJob,
}: ExportHubJobLifecycleProps) {
  const phase = exportJobPhase(status);
  const elapsed = useExportJobElapsed(phase === 'pending', startedAtMs);
  const canDownload = exportJobCanDownload(status);
  const canCancel = canCancelKind && exportJobCanCancel(status);
  const sizeSummary =
    phase === 'completed' ? formatExportJobRowSummary(rowLimit, bytes) : undefined;
  const failedJobMessage = exportHubJobErrorMessage(errorMessage);

  return (
    <div>
      <div>
        <ExportJobStatusBadge elapsed={elapsed} status={status} />
        {autoPolling && phase === 'pending' ? <span>Auto-refreshing every 3s</span> : null}
      </div>

      {phase === 'failed' && failedJobMessage ? (
        <ErrorBlock message={failedJobMessage} title="Export failed" />
      ) : null}

      <div>
        {canCancel ? (
          <Button
            disabled={exportBusy || !jobId.trim()}
            type="button"
            variant="destructive"
            onClick={onCancelJob}
          >
            {cancelling ? 'Cancelling...' : 'Cancel job'}
          </Button>
        ) : null}
        <Button
          disabled={exportBusy || !canDownload || !jobId.trim()}
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
        {sizeSummary ? <span>{sizeSummary}</span> : null}
      </div>
    </div>
  );
}
