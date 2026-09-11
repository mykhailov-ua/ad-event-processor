import { Button } from '@/components/ui/button';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { BentoSection } from '@/shell/bento_card';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';
import { EXPORT_HUB_KIND_LABELS } from '@/domains/exports/export_hub_catalog';
import { exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import type { ExportHubRecentJob } from '@/domains/exports/export_hub_recent';
import {
  exportJobCanDownloadFile,
  exportJobPhase,
  exportJobStatusDisplayLabel,
  formatExportJobRecentSummary,
  truncateExportJobInlineText,
} from '@/domains/exports/export_hub_job_status';

export type ExportHubRecentListProps = {
  jobs: ExportHubRecentJob[];
  activeJobId?: string;
  downloadingJobId?: string;
  rerunningJobId?: string;
  onSelectJob: (jobId: string) => void;
  onDownloadJob: (job: ExportHubRecentJob) => void;
  onRerunJob: (job: ExportHubRecentJob) => void;
};

export function ExportHubRecentList({
  jobs,
  activeJobId,
  downloadingJobId,
  rerunningJobId,
  onSelectJob,
  onDownloadJob,
  onRerunJob,
}: ExportHubRecentListProps) {
  if (jobs.length === 0) {
    return null;
  }

  return (
    <BentoSection data-testid="export-hub-recent" title="Recent exports">
      <p className={adminTypography.bodyMuted} data-role="recent-count">
        Last {jobs.length} job(s) in this browser session.
      </p>
      <ul className="grid gap-3" data-role="recent-list">
        {jobs.map((job) => {
          const active = job.jobId === activeJobId;
          const canDownload = exportJobCanDownloadFile(job.status, {
            destination: job.destination,
            spreadsheetUrl: job.spreadsheetUrl,
          });
          const canOpenSpreadsheet =
            exportJobPhase(job.status) === 'completed' && Boolean(job.spreadsheetUrl?.trim());
          const phase = exportJobPhase(job.status);
          const failed = phase === 'failed';
          const summary = formatExportJobRecentSummary(job.status, job.rowLimit, job.bytes);
          const errorMessage = failed ? exportHubJobErrorMessage(job.error) : undefined;
          const errorInline = errorMessage ? truncateExportJobInlineText(errorMessage) : undefined;
          const kindLabel = EXPORT_HUB_KIND_LABELS[job.kind];
          return (
            <li
              key={job.jobId}
              aria-current={active ? 'true' : undefined}
              className={cn(uiSurfaces.panel, 'grid gap-3')}
              data-role="recent-item"
              data-testid={`export-recent-${job.jobId}`}
            >
              <Button
                className="h-auto min-h-0 w-full justify-start whitespace-normal px-0 py-0 text-left font-normal hover:bg-transparent"
                type="button"
                variant="ghost"
                onClick={() => onSelectJob(job.jobId)}
              >
                <p className={adminTypography.label}>{kindLabel}</p>
                <p className={adminTypography.body}>{job.label}</p>
                <p className={adminTypography.monoData}>Job ID {job.jobId}</p>
                {job.customerId ? (
                  <p className={adminTypography.monoData}>Customer ID {job.customerId}</p>
                ) : null}
                {errorInline ? (
                  <p
                    className={adminTypography.bodyMuted}
                    title={errorInline.truncated ? errorInline.full : undefined}
                  >
                    {errorInline.display}
                  </p>
                ) : null}
                <p className={adminTypography.bodyMuted}>
                  {exportJobStatusDisplayLabel(job.status)}
                  {' / '}
                  {new Date(job.createdAt).toLocaleString()}
                  {summary ? ` / ${summary}` : null}
                </p>
              </Button>
              <div
                aria-label={`Actions for job ${job.jobId}`}
                className={adminSpacing.flex.buttonGroup}
                data-testid={`export-recent-actions-${job.jobId}`}
              >
                {canOpenSpreadsheet ? (
                  <Button asChild type="button" variant="outline">
                    <a href={job.spreadsheetUrl} rel="noopener noreferrer" target="_blank">
                      Open in Google Sheets
                    </a>
                  </Button>
                ) : null}
                {canDownload ? (
                  <Button
                    disabled={downloadingJobId === job.jobId}
                    type="button"
                    variant="outline"
                    onClick={() => onDownloadJob(job)}
                  >
                    {downloadingJobId === job.jobId ? 'Downloading...' : 'Download'}
                  </Button>
                ) : null}
                {job.kind === 'report' ? (
                  <Button
                    disabled={rerunningJobId === job.jobId}
                    type="button"
                    variant="outline"
                    onClick={() => onRerunJob(job)}
                  >
                    {rerunningJobId === job.jobId ? 'Re-running...' : 'Re-run'}
                  </Button>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
    </BentoSection>
  );
}
