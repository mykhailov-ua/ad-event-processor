import { Badge } from '@/components/ui/badge';
import {
  exportJobCanCancel,
  exportJobCanDownloadFile,
  exportJobPhase,
  formatExportJobRowSummary,
  normalizeExportJobStatus,
} from '@/domains/exports/export_hub_job_status';
import { exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import { useExportJobElapsed } from '@/domains/exports/use_export_job_elapsed';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { Button } from '@/components/ui/button';
import { ErrorBlock } from '@/shell/error_block';
import { BentoSection } from '@/shell/bento_card';

export type ExportHubJobLifecycleProps = {
  jobId: string;
  status: string | undefined;
  errorMessage?: string;
  bytes?: number;
  rowLimit?: number;
  startedAtMs?: number;
  destination?: 'download' | 'google_sheet';
  spreadsheetUrl?: string;
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
    return <Badge variant="outline">Running{elapsed ? ` / ${elapsed}` : '...'}</Badge>;
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
  destination,
  spreadsheetUrl,
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
  const canDownload = exportJobCanDownloadFile(status, { destination, spreadsheetUrl });
  const canOpenSpreadsheet = phase === 'completed' && Boolean(spreadsheetUrl?.trim());
  const canCancel = canCancelKind && exportJobCanCancel(status);
  const sizeSummary =
    phase === 'completed' ? formatExportJobRowSummary(rowLimit, bytes) : undefined;
  const failedJobMessage = exportHubJobErrorMessage(errorMessage);
  const hasStatus = Boolean(normalizeExportJobStatus(status));
  const trimmedJobId = jobId.trim();

  return (
    <BentoSection data-testid="export-job-lifecycle" title="Job status">
      {hasStatus ? (
        <div
          aria-label="Export job status"
          className={adminSpacing.flex.buttonGroup}
          data-testid="export-job-status"
        >
          <ExportJobStatusBadge elapsed={elapsed} status={status} />
          {autoPolling && phase === 'pending' ? (
            <span className={adminTypography.bodyMuted} data-role="auto-polling-hint">
              Auto-refreshing every 3s
            </span>
          ) : null}
          {sizeSummary ? (
            <span className={adminTypography.bodyMuted} data-role="size-summary">
              {sizeSummary}
            </span>
          ) : null}
        </div>
      ) : null}

      {phase === 'failed' && failedJobMessage ? (
        <div data-testid="export-job-error">
          <ErrorBlock message={failedJobMessage} title="Export failed" />
        </div>
      ) : null}

      <div
        aria-label="Export job actions"
        className={adminSpacing.flex.buttonGroup}
        data-testid="export-job-actions"
      >
        {canCancel ? (
          <Button
            data-testid="export-job-cancel"
            disabled={exportBusy || !trimmedJobId}
            type="button"
            variant="destructive"
            onClick={onCancelJob}
          >
            {cancelling ? 'Cancelling...' : 'Cancel job'}
          </Button>
        ) : null}
        {canOpenSpreadsheet ? (
          <Button asChild data-testid="export-job-open-sheet" type="button">
            <a href={spreadsheetUrl} rel="noopener noreferrer" target="_blank">
              Open in Google Sheets
            </a>
          </Button>
        ) : null}
        {canDownload ? (
          <Button
            data-testid="export-job-download"
            disabled={exportBusy || !trimmedJobId}
            title={sizeSummary ? `Download ${sizeSummary}` : 'Download completed export'}
            type="button"
            onClick={onDownloadJob}
          >
            {downloading ? 'Downloading...' : 'Download'}
          </Button>
        ) : null}
        <Button
          data-testid="export-job-refresh"
          disabled={exportBusy || !trimmedJobId}
          type="button"
          variant="outline"
          onClick={onPollJob}
        >
          {polling ? 'Polling...' : 'Refresh status'}
        </Button>
      </div>
    </BentoSection>
  );
}
