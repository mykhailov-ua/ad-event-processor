import { ExportHubCatalogPicker } from '@/domains/exports/export_hub_catalog_picker';
import type { ExportHubEntry } from '@/domains/exports/export_hub_catalog';
import { ExportHubJobLifecycle } from '@/domains/exports/export_hub_job_lifecycle';
import { ExportHubRecentList } from '@/domains/exports/export_hub_recent_list';
import type { ExportHubRecentJob } from '@/domains/exports/export_hub_recent';
import {
  EXPORT_HUB_ROW_LIMIT_DEFAULT,
  type ExportHubRowLimitBounds,
} from '@/domains/exports/export_hub_limits';
import { exportJobCanDownload } from '@/domains/exports/export_hub_job_status';
import type { BillingExportJob } from '@/api/types';
import type { ReportJobStatus } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  ExportHubCustomerIdHint,
  ExportHubJobIdHint,
} from '@/domains/exports/export_hub_field_hints';
import { PageChrome } from '@/shell/page_chrome';
import { FieldLabelWithHint } from '@/shell/field_label_hint';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';

export type ExportHubProps = {
  catalogEntries: ExportHubEntry[];
  catalogPickerValue: string;
  selectedKind: 'report' | 'billing' | 'audit';
  draftCustomerId: string;
  draftReportKey: string;
  draftFrom: string;
  draftTo: string;
  draftReportFormat: 'csv' | 'json' | '';
  draftBillingFormat: 'csv' | 'ndjson' | '';
  draftRedactPii: boolean;
  draftRowLimit: string;
  rowLimitBounds: ExportHubRowLimitBounds;
  draftJobId: string;
  job: ReportJobStatus | BillingExportJob | undefined;
  autoPolling: boolean;
  jobStartedAtMs?: number;
  recentJobs: ExportHubRecentJob[];
  creating: boolean;
  polling: boolean;
  downloading: boolean;
  cancelling: boolean;
  jobErrorMessage?: string;
  auditExportTruncated: boolean;
  onCatalogPickerChange: (value: string) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftReportKeyChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftReportFormatChange: (value: 'csv' | 'json') => void;
  onDraftBillingFormatChange: (value: 'csv' | 'ndjson') => void;
  onDraftRedactPiiChange: (value: boolean) => void;
  onDraftRowLimitChange: (value: string) => void;
  onDraftRowLimitBlur: () => void;
  onDraftJobIdChange: (value: string) => void;
  onRunExport: () => void;
  onPollJob: () => void;
  onCancelJob: () => void;
  onDownloadJob: () => void;
  onSelectRecentJob: (jobId: string) => void;
  onDownloadRecentJob: (job: ExportHubRecentJob) => void;
};

function jobStatusLabel(job: ReportJobStatus | BillingExportJob | undefined): string {
  return (job?.status ?? '').toString();
}

export function ExportHub({
  catalogEntries,
  catalogPickerValue,
  selectedKind,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftReportFormat,
  draftBillingFormat,
  draftRedactPii,
  draftRowLimit,
  rowLimitBounds,
  draftJobId,
  job,
  autoPolling,
  jobStartedAtMs,
  recentJobs,
  creating,
  polling,
  downloading,
  cancelling,
  jobErrorMessage,
  auditExportTruncated,
  onCatalogPickerChange,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftReportFormatChange,
  onDraftBillingFormatChange,
  onDraftRedactPiiChange,
  onDraftRowLimitChange,
  onDraftRowLimitBlur,
  onDraftJobIdChange,
  onRunExport,
  onPollJob,
  onCancelJob,
  onDownloadJob,
  onSelectRecentJob,
  onDownloadRecentJob,
}: ExportHubProps) {
  const jobStatus = jobStatusLabel(job);
  const canDownload = exportJobCanDownload(jobStatus);
  const exportBusy = creating || downloading || cancelling || (polling && !canDownload);
  const showAsyncJobPanel = selectedKind !== 'audit';
  const showRowLimit = selectedKind !== 'audit';
  const activeJobId = draftJobId.trim() || undefined;
  const runLabel =
    selectedKind === 'audit'
      ? creating
        ? 'Exporting...'
        : 'Download CSV'
      : creating
        ? 'Enqueueing...'
        : 'Start export';

  const runDisabled =
    exportBusy ||
    (selectedKind !== 'audit' && !draftCustomerId.trim()) ||
    (selectedKind === 'report' && !catalogPickerValue.trim());

  return (
    <PageChrome
      title="Exports"
      controlPanel={
        <div>
          <FilterPanel aria-label="Export form">
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <ExportHubCatalogPicker
                disabled={exportBusy}
                entries={catalogEntries}
                value={catalogPickerValue}
                onValueChange={onCatalogPickerChange}
              />

              {selectedKind !== 'audit' ? (
                <div>
                  <FieldLabelWithHint htmlFor="export-hub-customer-id" label="Customer ID">
                    <ExportHubCustomerIdHint />
                  </FieldLabelWithHint>
                  <Input
                    disabled={exportBusy}
                    id="export-hub-customer-id"
                    value={draftCustomerId}
                    onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                  />
                </div>
              ) : null}

              {selectedKind !== 'audit' ? (
                <>
                  <DatetimePicker
                    disabled={exportBusy}
                    id="export-hub-from"
                    label="From"
                    value={draftFrom}
                    onChange={onDraftFromChange}
                  />
                  <DatetimePicker
                    disabled={exportBusy}
                    id="export-hub-to"
                    label="To"
                    value={draftTo}
                    onChange={onDraftToChange}
                  />
                </>
              ) : null}

              {showRowLimit ? (
                <FilterField htmlFor="export-hub-row-limit" label="Row limit">
                  <Input
                    aria-describedby="export-hub-row-limit-max"
                    disabled={exportBusy}
                    id="export-hub-row-limit"
                    inputMode="numeric"
                    max={rowLimitBounds.max}
                    min={rowLimitBounds.min}
                    placeholder={`Default ${EXPORT_HUB_ROW_LIMIT_DEFAULT.toLocaleString()}`}
                    title={`Empty uses ${EXPORT_HUB_ROW_LIMIT_DEFAULT.toLocaleString()} rows. Hard max ${rowLimitBounds.max.toLocaleString()}.`}
                    type="number"
                    value={draftRowLimit}
                    onBlur={onDraftRowLimitBlur}
                    onChange={(event) => onDraftRowLimitChange(event.target.value)}
                  />
                  <span hidden id="export-hub-row-limit-max">
                    Maximum {rowLimitBounds.max.toLocaleString()} rows for your access tier
                  </span>
                </FilterField>
              ) : null}

              {selectedKind === 'report' ? (
                <FilterField htmlFor="export-hub-report-format" label="Format">
                  <Select
                    disabled={exportBusy}
                    value={draftReportFormat || undefined}
                    onValueChange={(value) =>
                      onDraftReportFormatChange(value as 'csv' | 'json')
                    }
                  >
                    <SelectTrigger id="export-hub-report-format">
                      <SelectValue placeholder="Select format" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="csv">CSV</SelectItem>
                      <SelectItem value="json">JSON</SelectItem>
                    </SelectContent>
                  </Select>
                </FilterField>
              ) : null}

              {selectedKind === 'billing' ? (
                <FilterField htmlFor="export-hub-billing-format" label="Format">
                  <Select
                    disabled={exportBusy}
                    value={draftBillingFormat || undefined}
                    onValueChange={(value) =>
                      onDraftBillingFormatChange(value as 'csv' | 'ndjson')
                    }
                  >
                    <SelectTrigger id="export-hub-billing-format">
                      <SelectValue placeholder="Select format" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="csv">CSV</SelectItem>
                      <SelectItem value="ndjson">NDJSON</SelectItem>
                    </SelectContent>
                  </Select>
                </FilterField>
              ) : null}

              {selectedKind === 'audit' ? (
                <div>
                  <Checkbox
                    checked={draftRedactPii}
                    id="export-hub-redact-pii"
                    onCheckedChange={onDraftRedactPiiChange}
                  />
                  <Label htmlFor="export-hub-redact-pii">Redact PII</Label>
                </div>
              ) : null}

              <Button disabled={runDisabled} type="button" onClick={onRunExport}>
                {runLabel}
              </Button>
            </DirectoryFilterForm>
          </FilterPanel>

          {showAsyncJobPanel ? (
            <FilterPanel aria-label="Job controls">
              <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
                <div>
                  <FieldLabelWithHint htmlFor="export-hub-job-id" label="Job ID">
                    <ExportHubJobIdHint />
                  </FieldLabelWithHint>
                  <Input
                    id="export-hub-job-id"
                    value={draftJobId}
                    onChange={(event) => onDraftJobIdChange(event.target.value)}
                  />
                </div>

                <ExportHubJobLifecycle
                  autoPolling={autoPolling}
                  bytes={job?.bytes}
                  canCancelKind={selectedKind === 'report'}
                  cancelling={cancelling}
                  downloading={downloading}
                  errorMessage={jobErrorMessage ?? job?.error ?? undefined}
                  exportBusy={exportBusy}
                  jobId={draftJobId}
                  rowLimit={
                    job && 'row_limit' in job && typeof job.row_limit === 'number'
                      ? job.row_limit
                      : undefined
                  }
                  startedAtMs={jobStartedAtMs}
                  status={jobStatus}
                  onCancelJob={onCancelJob}
                  onDownloadJob={onDownloadJob}
                />

                <Button
                  disabled={exportBusy || !draftJobId.trim()}
                  type="button"
                  variant="outline"
                  onClick={onPollJob}
                >
                  {polling ? 'Polling...' : 'Refresh status'}
                </Button>
              </DirectoryFilterForm>
            </FilterPanel>
          ) : null}
        </div>
      }
    >
      <div>
        {auditExportTruncated ? (
          <p role="status">
            Audit export was truncated. Use audit list filters or request a smaller window if
            needed.
          </p>
        ) : null}

        <ExportHubRecentList
          activeJobId={activeJobId}
          downloadingJobId={downloading ? activeJobId : undefined}
          jobs={recentJobs}
          onDownloadJob={onDownloadRecentJob}
          onSelectJob={onSelectRecentJob}
        />
      </div>
    </PageChrome>
  );
}
