import { Button } from '@/components/ui/button';
import { adminTypography } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';
import { EXPORT_HUB_KIND_LABELS } from '@/domains/exports/export_hub_catalog';
import { exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import type { ExportHubRecentJob } from '@/domains/exports/export_hub_recent';
import {
  exportJobCanDownload,
  exportJobPhase,
  exportJobStatusDisplayLabel,
  formatExportJobRecentSummary,
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
    <section aria-label="Recent exports" className={uiSurfaces.panel}>
      <h2 className={adminTypography.sectionTitle}>Recent exports</h2>
      <p className={adminTypography.bodyMuted}>Last {jobs.length} job(s) in this browser session.</p>
      <ul className={cn('m-0 list-none p-0', uiSurfaces.directoryStack)}>
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
            <li
              key={job.jobId}
              aria-current={active ? 'true' : undefined}
              className={cn(
                'grid gap-2 border border-border p-3',
                active && 'border-primary/40 bg-admin-selection'
              )}
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
                  {' · '}
                  {new Date(job.createdAt).toLocaleString()}
                  {summary ? ` · ${summary}` : null}
                </p>
              </Button>
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
