import { ExportHubCatalogPicker } from '@/domains/exports/export_hub_catalog_picker';
import { ExportsNav } from '@/domains/exports/exports_nav';
import type { ExportHubEntry } from '@/domains/exports/export_hub_catalog';
import { ExportHubCompareFields } from '@/domains/exports/export_hub_compare_fields';
import { ExportHubJobLifecycle } from '@/domains/exports/export_hub_job_lifecycle';
import {
  ExportHubNotifyFields,
  type ExportHubNotifyChannel,
} from '@/domains/exports/export_hub_notify_fields';
import { ExportHubNotifications } from '@/domains/exports/export_hub_notifications';
import { ExportHubRecentList } from '@/domains/exports/export_hub_recent_list';
import { ExportHubSavedViews } from '@/domains/exports/export_hub_saved_views';
import type { SavedView } from '@/api/types';
import type { ExportHubRecentJob, ExportHubDestination } from '@/domains/exports/export_hub_recent';
import {
  EXPORT_HUB_ROW_LIMIT_DEFAULT,
  type ExportHubRowLimitBounds,
} from '@/domains/exports/export_hub_limits';
import { exportJobCanDownloadFile } from '@/domains/exports/export_hub_job_status';
import type { ExportHubReportFormat } from '@/domains/exports/export_hub_report_formats';
import type { GoogleSheetsIntegrationStatus } from '@/api/integrations_api';
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
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import { PageSectionStack } from '@/shell/page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import { FieldLabelWithHint } from '@/shell/field_label_hint';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS,
} from '@/shell/filter_panel';
import { adminKit } from '@/lib/admin_kit';
import { adminSpacing, adminTypography, opsControlPanelClass } from '@/lib/admin_spacing';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { cn } from '@/lib/utils';
import { Link } from 'react-router-dom';

const REPORT_FORMAT_LABELS: Record<ExportHubReportFormat, string> = {
  csv: 'CSV',
  xlsx: 'Excel (XLSX)',
  json: 'JSON',
  zip: 'ZIP',
};

export type ExportHubCampaignToggleField =
  | 'silent_reject_enabled'
  | 'accept_lang_geo_enabled'
  | 'json_serialization_enabled';

export type ExportHubProps = {
  catalogEntries: ExportHubEntry[];
  catalogError?: Error;
  catalogFetching: boolean;
  catalogHasSnapshot: boolean;
  onRefreshCatalog: () => void;
  catalogPickerValue: string;
  selectedKind: 'report' | 'billing' | 'audit' | 'directory';
  directoryLinkHref?: string;
  selectedEntryDescription?: string;
  draftCustomerId: string;
  draftReportKey: string;
  draftFrom: string;
  draftTo: string;
  draftCompareFrom: string;
  draftCompareTo: string;
  draftNotifyChannel: ExportHubNotifyChannel;
  draftNotifyEmail: string;
  draftNotifyWebhookUrl: string;
  draftReportFormat: ExportHubReportFormat | '';
  reportFormatOptions: ExportHubReportFormat[];
  draftDestination: ExportHubDestination;
  draftCampaignToggleCampaignId: string;
  draftCampaignToggleField: ExportHubCampaignToggleField | '';
  draftCampaignToggleAt: string;
  draftCampaignToggleWindowHours: string;
  draftLayerDesyncCount: string;
  draftSpreadsheetId: string;
  draftSheetTitle: string;
  googleSheetsStatus: GoogleSheetsIntegrationStatus | undefined;
  googleSheetsStatusError: Error | undefined;
  googleSheetsStatusFetching: boolean;
  jobSpreadsheetUrl?: string;
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
  rerunningJobId?: string;
  polling: boolean;
  downloading: boolean;
  cancelling: boolean;
  jobErrorMessage?: string;
  auditExportTruncated: boolean;
  formValidationError?: AdminValidationError;
  onCatalogPickerChange: (value: string) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftReportKeyChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCompareFromChange: (value: string) => void;
  onDraftCompareToChange: (value: string) => void;
  onDraftNotifyChannelChange: (value: ExportHubNotifyChannel) => void;
  onDraftNotifyEmailChange: (value: string) => void;
  onDraftNotifyWebhookUrlChange: (value: string) => void;
  onDraftReportFormatChange: (value: ExportHubReportFormat) => void;
  onDraftDestinationChange: (value: ExportHubDestination) => void;
  onDraftCampaignToggleCampaignIdChange: (value: string) => void;
  onDraftCampaignToggleFieldChange: (value: ExportHubCampaignToggleField) => void;
  onDraftCampaignToggleAtChange: (value: string) => void;
  onDraftCampaignToggleWindowHoursChange: (value: string) => void;
  onDraftLayerDesyncCountChange: (value: string) => void;
  onDraftSpreadsheetIdChange: (value: string) => void;
  onDraftSheetTitleChange: (value: string) => void;
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
  onRerunRecentJob: (job: ExportHubRecentJob) => void;
  savedViews: SavedView[];
  savedViewsHasSnapshot: boolean;
  savedViewsError?: Error;
  savedViewsFetching: boolean;
  canManagePresets: boolean;
  canExportPreset: boolean;
  presetName: string;
  selectedViewId: string;
  savingPreset: boolean;
  exportingViewId?: string;
  onPresetNameChange: (value: string) => void;
  onSelectedViewIdChange: (value: string) => void;
  onLoadSavedView: () => void;
  onSaveSavedView: () => void;
  onDeleteSavedView: () => void;
  onExportSavedView: () => void;
};

function jobStatusLabel(job: ReportJobStatus | BillingExportJob | undefined): string {
  return (job?.status ?? '').toString();
}

export function ExportHub({
  catalogEntries,
  catalogError,
  catalogFetching,
  catalogHasSnapshot,
  onRefreshCatalog,
  catalogPickerValue,
  selectedKind,
  directoryLinkHref,
  selectedEntryDescription,
  draftCustomerId,
  draftReportKey,
  draftFrom,
  draftTo,
  draftCompareFrom,
  draftCompareTo,
  draftNotifyChannel,
  draftNotifyEmail,
  draftNotifyWebhookUrl,
  draftReportFormat,
  reportFormatOptions,
  draftDestination,
  draftCampaignToggleCampaignId,
  draftCampaignToggleField,
  draftCampaignToggleAt,
  draftCampaignToggleWindowHours,
  draftLayerDesyncCount,
  draftSpreadsheetId,
  draftSheetTitle,
  googleSheetsStatus,
  googleSheetsStatusError,
  googleSheetsStatusFetching,
  jobSpreadsheetUrl,
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
  rerunningJobId,
  polling,
  downloading,
  cancelling,
  jobErrorMessage,
  auditExportTruncated,
  formValidationError,
  onCatalogPickerChange,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCompareFromChange,
  onDraftCompareToChange,
  onDraftNotifyChannelChange,
  onDraftNotifyEmailChange,
  onDraftNotifyWebhookUrlChange,
  onDraftReportFormatChange,
  onDraftDestinationChange,
  onDraftCampaignToggleCampaignIdChange,
  onDraftCampaignToggleFieldChange,
  onDraftCampaignToggleAtChange,
  onDraftCampaignToggleWindowHoursChange,
  onDraftLayerDesyncCountChange,
  onDraftSpreadsheetIdChange,
  onDraftSheetTitleChange,
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
  onRerunRecentJob,
  savedViews,
  savedViewsHasSnapshot,
  savedViewsError,
  savedViewsFetching,
  canManagePresets,
  canExportPreset,
  presetName,
  selectedViewId,
  savingPreset,
  exportingViewId,
  onPresetNameChange,
  onSelectedViewIdChange,
  onLoadSavedView,
  onSaveSavedView,
  onDeleteSavedView,
  onExportSavedView,
}: ExportHubProps) {
  const jobStatus = jobStatusLabel(job);
  const jobDestination =
    job && 'destination' in job && job.destination === 'google_sheet'
      ? 'google_sheet'
      : draftDestination;
  const spreadsheetUrl =
    jobSpreadsheetUrl ??
    (job && 'spreadsheet_url' in job ? job.spreadsheet_url?.trim() : undefined);
  const canDownload = exportJobCanDownloadFile(jobStatus, {
    destination: jobDestination,
    spreadsheetUrl,
  });
  const exportBusy = creating || downloading || cancelling || (polling && !canDownload);
  const isDirectoryExport = selectedKind === 'directory';
  const showAsyncJobPanel = selectedKind !== 'audit' && !isDirectoryExport;
  const showRowLimit = selectedKind !== 'audit' && !isDirectoryExport;
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
  const catalogRetryButton = (
    <Button
      data-testid="export-hub-catalog-retry"
      disabled={catalogFetching}
      type="button"
      variant="outline"
      onClick={onRefreshCatalog}
    >
      {catalogFetching ? 'Retrying...' : 'Retry'}
    </Button>
  );

  return (
    <DirectoryPageShell
      actions={<ExportHubNotifications onOpenJob={onSelectRecentJob} />}
      blockingErrorFooter={<div data-testid="export-hub-catalog-error">{catalogRetryButton}</div>}
      blockingErrorTitle="Could not load export catalog"
      controlPanel={
        <div className={opsControlPanelClass}>
          <ExportsNav />
          <FilterPanel aria-label="Export form">
            {formValidationError ? (
              <ErrorBlock error={formValidationError} title="Check export fields" />
            ) : null}
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <ExportHubCatalogPicker
                disabled={exportBusy}
                entries={catalogEntries}
                value={catalogPickerValue}
                onValueChange={onCatalogPickerChange}
              />

              {isDirectoryExport && directoryLinkHref ? (
                <div className={cn('col-span-full grid min-w-0', adminSpacing.gap.md)}>
                  <StubBanner
                    message={
                      selectedEntryDescription ??
                      'Open the directory page and use Export CSV in the toolbar.'
                    }
                    title="Directory export"
                  />
                  <div className={adminSpacing.flex.buttonGroup}>
                    <Button asChild type="button" variant="brand">
                      <Link to={directoryLinkHref}>Open campaigns directory</Link>
                    </Button>
                  </div>
                </div>
              ) : null}

              {selectedKind !== 'audit' && !isDirectoryExport ? (
                <div className={cn('grid min-w-0', adminKit.fieldLabelGap)}>
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

              {selectedKind !== 'audit' && !isDirectoryExport ? (
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
                  {selectedKind === 'report' ? (
                    <ExportHubCompareFields
                      compareFrom={draftCompareFrom}
                      compareTo={draftCompareTo}
                      disabled={exportBusy}
                      onCompareFromChange={onDraftCompareFromChange}
                      onCompareToChange={onDraftCompareToChange}
                    />
                  ) : null}
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
                <>
                  <FilterField htmlFor="export-hub-report-format" label="Format">
                    <Select
                      disabled={exportBusy}
                      value={draftReportFormat || undefined}
                      onValueChange={(value) =>
                        onDraftReportFormatChange(value as ExportHubReportFormat)
                      }
                    >
                      <SelectTrigger id="export-hub-report-format">
                        <SelectValue placeholder="Select format" />
                      </SelectTrigger>
                      <SelectContent>
                        {reportFormatOptions.map((format) => (
                          <SelectItem key={format} value={format}>
                            {REPORT_FORMAT_LABELS[format]}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FilterField>

                  <FilterField htmlFor="export-hub-destination" label="Destination">
                    <Select
                      disabled={exportBusy}
                      value={draftDestination}
                      onValueChange={(value) =>
                        onDraftDestinationChange(value as ExportHubDestination)
                      }
                    >
                      <SelectTrigger id="export-hub-destination">
                        <SelectValue placeholder="Select destination" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="download">Download file</SelectItem>
                        <SelectItem value="google_sheet">Google Sheet</SelectItem>
                      </SelectContent>
                    </Select>
                  </FilterField>
                  <ExportHubNotifyFields
                    channel={draftNotifyChannel}
                    disabled={exportBusy}
                    email={draftNotifyEmail}
                    webhookUrl={draftNotifyWebhookUrl}
                    onChannelChange={onDraftNotifyChannelChange}
                    onEmailChange={onDraftNotifyEmailChange}
                    onWebhookUrlChange={onDraftNotifyWebhookUrlChange}
                  />
                </>
              ) : null}

              {selectedKind === 'report' && draftReportKey === 'campaign-toggle-cohort' ? (
                <>
                  <FilterField htmlFor="export-hub-toggle-campaign-id" label="Campaign ID">
                    <Input
                      disabled={exportBusy}
                      id="export-hub-toggle-campaign-id"
                      value={draftCampaignToggleCampaignId}
                      onChange={(event) =>
                        onDraftCampaignToggleCampaignIdChange(event.target.value)
                      }
                    />
                  </FilterField>
                  <FilterField htmlFor="export-hub-toggle-field" label="Toggle field">
                    <Select
                      disabled={exportBusy}
                      value={draftCampaignToggleField || undefined}
                      onValueChange={(value) =>
                        onDraftCampaignToggleFieldChange(value as ExportHubCampaignToggleField)
                      }
                    >
                      <SelectTrigger id="export-hub-toggle-field">
                        <SelectValue placeholder="Select toggle field" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="silent_reject_enabled">silent_reject_enabled</SelectItem>
                        <SelectItem value="accept_lang_geo_enabled">
                          accept_lang_geo_enabled
                        </SelectItem>
                        <SelectItem value="json_serialization_enabled">
                          json_serialization_enabled
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </FilterField>
                  <DatetimePicker
                    disabled={exportBusy}
                    id="export-hub-toggle-at"
                    label="Toggle at"
                    value={draftCampaignToggleAt}
                    onChange={onDraftCampaignToggleAtChange}
                  />
                  <FilterField htmlFor="export-hub-toggle-window-hours" label="Window hours">
                    <Input
                      disabled={exportBusy}
                      id="export-hub-toggle-window-hours"
                      inputMode="numeric"
                      max={168}
                      min={1}
                      placeholder="Default 72"
                      type="number"
                      value={draftCampaignToggleWindowHours}
                      onChange={(event) =>
                        onDraftCampaignToggleWindowHoursChange(event.target.value)
                      }
                    />
                  </FilterField>
                </>
              ) : null}

              {selectedKind === 'report' && draftReportKey === 'layer-desync-drilldown' ? (
                <FilterField htmlFor="export-hub-layer-desync-count" label="Layer desync count">
                  <Input
                    disabled={exportBusy}
                    id="export-hub-layer-desync-count"
                    inputMode="numeric"
                    max={255}
                    min={1}
                    placeholder="Default 2"
                    type="number"
                    value={draftLayerDesyncCount}
                    onChange={(event) => onDraftLayerDesyncCountChange(event.target.value)}
                  />
                </FilterField>
              ) : null}

              {selectedKind === 'report' && draftDestination === 'google_sheet' ? (
                <>
                  {googleSheetsStatusError ? (
                    <ErrorBlock
                      error={googleSheetsStatusError}
                      title="Google Sheets status unavailable"
                    />
                  ) : null}
                  {!googleSheetsStatusFetching && !googleSheetsStatus?.connected ? (
                    <div className={cn('grid min-w-0', adminSpacing.gap.sm)}>
                      <StubBanner
                        message="Connect your Google account to push export rows into a spreadsheet."
                        title="Google Sheets not connected"
                      />
                      <Link className="text-primary underline" to="/integrations/google-sheets">
                        Connect Google Sheets
                      </Link>
                    </div>
                  ) : null}
                  {googleSheetsStatus?.connected ? (
                    <p className={adminTypography.bodyMuted}>
                      Connected
                      {googleSheetsStatus.account_email
                        ? ` as ${googleSheetsStatus.account_email}`
                        : ''}
                      .
                    </p>
                  ) : null}
                  <FilterField htmlFor="export-hub-spreadsheet-id" label="Spreadsheet ID (append)">
                    <Input
                      disabled={exportBusy}
                      id="export-hub-spreadsheet-id"
                      placeholder="Optional for append mode"
                      value={draftSpreadsheetId}
                      onChange={(event) => onDraftSpreadsheetIdChange(event.target.value)}
                    />
                  </FilterField>
                  <FilterField htmlFor="export-hub-sheet-title" label="Sheet title">
                    <Input
                      disabled={exportBusy}
                      id="export-hub-sheet-title"
                      placeholder="Optional"
                      value={draftSheetTitle}
                      onChange={(event) => onDraftSheetTitleChange(event.target.value)}
                    />
                  </FilterField>
                </>
              ) : null}

              {selectedKind === 'billing' ? (
                <FilterField htmlFor="export-hub-billing-format" label="Format">
                  <Select
                    disabled={exportBusy}
                    value={draftBillingFormat || undefined}
                    onValueChange={(value) => onDraftBillingFormatChange(value as 'csv' | 'ndjson')}
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
                <div className={cn(adminSpacing.flex.buttonGroup, 'items-center self-end')}>
                  <Checkbox
                    checked={draftRedactPii}
                    id="export-hub-redact-pii"
                    onCheckedChange={onDraftRedactPiiChange}
                  />
                  <Label htmlFor="export-hub-redact-pii">Redact PII</Label>
                </div>
              ) : null}

              {!isDirectoryExport ? (
                <Button disabled={runDisabled} type="button" onClick={onRunExport}>
                  {runLabel}
                </Button>
              ) : null}
            </DirectoryFilterForm>
          </FilterPanel>

          {showAsyncJobPanel ? (
            <FilterPanel aria-label="Job controls">
              <div className={INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS}>
                <div className={cn('grid min-w-0', adminKit.fieldLabelGap)}>
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
                  destination={jobDestination}
                  downloading={downloading}
                  errorMessage={jobErrorMessage ?? job?.error ?? undefined}
                  exportBusy={exportBusy}
                  jobId={draftJobId}
                  polling={polling}
                  rowLimit={
                    job && 'row_limit' in job && typeof job.row_limit === 'number'
                      ? job.row_limit
                      : undefined
                  }
                  spreadsheetUrl={spreadsheetUrl}
                  startedAtMs={jobStartedAtMs}
                  status={jobStatus}
                  onCancelJob={onCancelJob}
                  onDownloadJob={onDownloadJob}
                  onPollJob={onPollJob}
                />
              </div>
            </FilterPanel>
          ) : null}
        </div>
      }
      description="Async report and billing exports. Full tabular reports run as jobs; download when complete."
      fetchState={{
        fetching: catalogFetching,
        error: catalogError,
        hasSnapshot: catalogHasSnapshot,
      }}
      refreshErrorFooter={
        <div data-testid="export-hub-catalog-refresh-error">{catalogRetryButton}</div>
      }
      refreshErrorTitle="Export catalog refresh failed"
      title="Exports"
    >
      <PageSectionStack>
        {auditExportTruncated ? (
          <p className={adminTypography.bodyMuted} role="status">
            Audit export was truncated. Use audit list filters or request a smaller window if
            needed.
          </p>
        ) : null}

        <ExportHubSavedViews
          canExport={canExportPreset}
          canManage={canManagePresets}
          customerId={draftCustomerId}
          exportingViewId={exportingViewId}
          presetName={presetName}
          saving={savingPreset}
          selectedViewId={selectedViewId}
          views={savedViews}
          viewsHasSnapshot={savedViewsHasSnapshot}
          viewsError={savedViewsError}
          viewsFetching={savedViewsFetching}
          onDeleteView={onDeleteSavedView}
          onExportView={onExportSavedView}
          onLoadView={onLoadSavedView}
          onPresetNameChange={onPresetNameChange}
          onSaveView={onSaveSavedView}
          onSelectedViewIdChange={onSelectedViewIdChange}
        />

        <ExportHubRecentList
          activeJobId={activeJobId}
          downloadingJobId={downloading ? activeJobId : undefined}
          jobs={recentJobs}
          rerunningJobId={rerunningJobId}
          onDownloadJob={onDownloadRecentJob}
          onRerunJob={onRerunRecentJob}
          onSelectJob={onSelectRecentJob}
        />
      </PageSectionStack>
    </DirectoryPageShell>
  );
}
