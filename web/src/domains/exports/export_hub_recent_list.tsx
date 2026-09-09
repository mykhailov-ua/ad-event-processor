import { Button } from '@/components/ui/button';
import { EXPORT_HUB_KIND_LABELS } from '@/domains/exports/export_hub_catalog';
import { exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import type { ExportHubRecentJob } from '@/domains/exports/export_hub_recent';
import {
  exportJobCanDownload,
  exportJobPhase,
  formatExportJobRecentSummary,
  normalizeExportJobStatus,
  truncateExportJobInlineText,
} from '@/domains/exports/export_hub_job_status';

export type ExportHubRecentListProps = {
  jobs: ExportHubRecentJob[];
  activeJobId?: string;
  downloadingJobId?: string;
  onSelectJob: (jobId: string) => void;
  onDownloadJob: (job: ExportHubRecentJob) => void;
};

export function ExportHubRecentList({
  jobs,
  activeJobId,
  downloadingJobId,
  onSelectJob,
  onDownloadJob,
}: ExportHubRecentListProps) {
  if (jobs.length === 0) {
    return null;
  }

  return (
    <section aria-label="Recent exports">
      <h2>Recent exports</h2>
      <p>Last {jobs.length} job(s) in this browser session.</p>
      <ul>
        {jobs.map((job) => {
          const active = job.jobId === activeJobId;
          const canDownload = exportJobCanDownload(job.status);
          const phase = exportJobPhase(job.status);
          const failed = phase === 'failed';
          const summary = formatExportJobRecentSummary(job.status, job.rowLimit, job.bytes);
          const errorMessage = failed ? exportHubJobErrorMessage(job.error) : undefined;
          const errorInline = errorMessage ? truncateExportJobInlineText(errorMessage) : undefined;
          const kindLabel = EXPORT_HUB_KIND_LABELS[job.kind];
          return (
            <li key={job.jobId} aria-current={active ? 'true' : undefined}>
              <button type="button" onClick={() => onSelectJob(job.jobId)}>
                <p>{kindLabel}</p>
                <p>{job.label}</p>
                <p>Job ID {job.jobId}</p>
                {job.customerId ? <p>Customer ID {job.customerId}</p> : null}
                {errorInline ? (
                  <p title={errorInline.truncated ? errorInline.full : undefined}>
                    {errorInline.display}
                  </p>
                ) : null}
                <p>
                  {normalizeExportJobStatus(job.status) || 'unknown'}
                  {' · '}
                  {new Date(job.createdAt).toLocaleString()}
                  {summary ? ` · ${summary}` : null}
                </p>
              </button>
              <Button
                disabled={!canDownload || downloadingJobId === job.jobId}
                type="button"
                variant="outline"
                onClick={() => onDownloadJob(job)}
              >
                {downloadingJobId === job.jobId ? 'Downloading...' : 'Download'}
              </Button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
