import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import { toast } from 'sonner';

import { exportAuditCsv } from '@/api/audit_api';
import {
  createBillingExportJob,
  downloadBillingExportJob,
  getBillingExportJob,
} from '@/api/billing_api';
import { getGoogleSheetsStatus } from '@/api/integrations_api';
import {
  cancelReportJob,
  createReportJob,
  downloadReportJob,
  getReportJob,
  rerunReportJob,
} from '@/api/reports_api';
import type { SavedView } from '@/api/types';
import {
  createSavedView,
  deleteSavedView,
  exportSavedView,
  listSavedViews,
  updateSavedView,
} from '@/api/views_api';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { isUuidLike } from '@/lib/customer_label';
import {
  exportHubCatalogOptionValue,
  exportHubEntriesFromCatalog,
  findExportHubEntry,
  resolveExportHubCatalogValue,
  type ExportHubEntry,
  type ExportHubKind,
} from '@/domains/exports/export_hub_catalog';
import type { ExportHubCampaignToggleField } from '@/domains/exports/export_hub';
import {
  clampExportHubRowLimit,
  normalizeExportHubRowLimitDraft,
  parseExportHubRowLimitDraft,
  resolveExportHubRowLimitBounds,
} from '@/domains/exports/export_hub_limits';
import {
  applyExportHubSavedViewSpec,
  buildExportHubSavedViewSpec,
  resolveSavedViewReportKey,
  type ExportHubSavedViewApplyPatch,
} from '@/domains/exports/export_hub_saved_view_spec';
import { exportHubErrorMessage, exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import {
  exportJobPhase,
  exportJobSpreadsheetUrl,
  normalizeExportJobStatus,
} from '@/domains/exports/export_hub_job_status';
import type { ExportHubDestination } from '@/domains/exports/export_hub_recent';
import type { ExportHubNotifyChannel } from '@/domains/exports/export_hub_notify_fields';
import {
  listExportHubRecentJobs,
  patchExportHubRecentJob,
  upsertExportHubRecentJob,
  type ExportHubRecentJob,
} from '@/domains/exports/export_hub_recent';
import {
  resolveExportHubReportFormats,
  type ExportHubReportFormat,
} from '@/domains/exports/export_hub_report_formats';
import {
  fetchReportCatalogCached,
  invalidateReportCatalogCache,
} from '@/lib/report_catalog_cache';
import { resolveReportDisplayTitle } from '@/lib/report_paths';
import { confirmDestructiveAction } from '@/lib/mutation_audit';
import { sessionHasPermission } from '@/lib/session_permissions';
import {
  type AdminValidationError,
  requireDateRange,
  requireInteger,
  requireNonEmpty,
  toastValidationError,
  validationError,
  type ValidationResult,
} from '@/lib/admin_validation_error';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

export type ExportHubStatusPhase = 'idle' | 'pending';

const EXPORT_PENDING_TOAST_ID = 'export-hub-pending';

function resolveExportPendingToastMessage(
  creating: boolean,
  downloading: boolean,
  cancelling: boolean
): string {
  if (downloading) {
    return 'Downloading export...';
  }
  if (creating) {
    return 'Enqueueing export...';
  }
  if (cancelling) {
    return 'Cancelling export...';
  }
  return 'Server is preparing your export...';
}

const EXPORT_HUB_POLL_INTERVAL_MS = 2500;

type ReportFormat = ExportHubReportFormat;
type BillingFormat = 'csv' | 'ndjson';
type CampaignToggleField = ExportHubCampaignToggleField;

function parseDestination(value: string | null): ExportHubDestination {
  return value === 'google_sheet' ? 'google_sheet' : 'download';
}

function reportDownloadExtension(format: string | undefined): string {
  if (format === 'xlsx' || format === 'json' || format === 'zip' || format === 'csv') {
    return format;
  }
  return 'csv';
}

function parseKind(value: string | null): ExportHubKind | undefined {
  if (value === 'report' || value === 'billing' || value === 'audit' || value === 'directory') {
    return value;
  }
  return undefined;
}

function isJobReadyStatus(status: string): boolean {
  return exportJobPhase(status) === 'completed';
}

function isJobPendingStatus(status: string): boolean {
  return exportJobPhase(status) === 'pending';
}

function recentJobLabel(
  selectedKind: ExportHubKind,
  selectedEntry: ExportHubEntry | undefined,
  reportKey: string
): string {
  if (selectedKind === 'billing') {
    return 'Billing export';
  }
  if (selectedKind === 'audit') {
    return 'Audit CSV';
  }
  return selectedEntry?.title ?? reportKey;
}

export function useExportHubPageWorkspace() {
  const [searchParams, { replaceSearchParams }] = useTransitionSearchParams();
  const { session, user } = useSession();
  const [catalogRefreshToken, setCatalogRefreshToken] = useState(0);
  const {
    data: catalog,
    error: catalogError,
    fetching: catalogFetching,
  } = useResource(
    (signal) => fetchReportCatalogCached(signal),
    [catalogRefreshToken]
  );
  const catalogHasSnapshot = catalog != null;

  const catalogEntries = useMemo(
    () => exportHubEntriesFromCatalog(catalog?.rows),
    [catalog?.rows]
  );

  const entryId = searchParams.get('entry') ?? '';
  const kindFromUrl = parseKind(searchParams.get('kind'));
  const jobId = searchParams.get('job_id') ?? '';

  const selectedEntry = useMemo(() => {
    if (entryId) {
      return findExportHubEntry(entryId, catalog?.rows);
    }
    const reportKey = searchParams.get('report_key')?.trim();
    if (reportKey) {
      return (
        catalogEntries.find(
          (entry) => entry.kind === 'report' && entry.reportKey === reportKey
        )         ?? {
          id: `custom-${reportKey}`,
          title: resolveReportDisplayTitle(reportKey),
          description: 'Custom report key from catalog search.',
          kind: 'report' as const,
          defaultFormat: 'csv',
          reportKey,
        }
      );
    }
    if (kindFromUrl) {
      return catalogEntries.find((entry) => entry.kind === kindFromUrl);
    }
    return catalogEntries[0];
  }, [catalog?.rows, catalogEntries, entryId, kindFromUrl, searchParams]);

  const selectedKind = selectedEntry?.kind ?? kindFromUrl ?? 'report';

  const rowLimitBounds = useMemo(
    () =>
      resolveExportHubRowLimitBounds({
        licenseGated: Boolean(
          selectedEntry?.kind === 'report' &&
            catalog?.rows?.find((row) => row.key === selectedEntry.reportKey)?.license_gated
        ),
      }),
    [catalog?.rows, selectedEntry]
  );

  const selectedCatalogRow = useMemo(
    () =>
      selectedEntry?.kind === 'report' && selectedEntry.reportKey
        ? catalog?.rows?.find((row) => row.key === selectedEntry.reportKey)
        : undefined,
    [catalog?.rows, selectedEntry]
  );

  const [catalogPickerValue, setCatalogPickerValue] = useState('');
  const [draftCustomerId, setDraftCustomerId] = useState(
    searchParams.get('customer_id') ?? session?.default_customer_id ?? ''
  );
  const [draftReportKey, setDraftReportKey] = useState(
    searchParams.get('report_key') ?? selectedEntry?.reportKey ?? 'placements'
  );
  const urlFrom = searchParams.get('from');
  const urlTo = searchParams.get('to');
  const urlFormat = searchParams.get('format');
  const [draftFrom, setDraftFrom] = useState(() =>
    urlFrom ? toDatetimeLocalValue(urlFrom) : ''
  );
  const [draftTo, setDraftTo] = useState(() =>
    urlTo ? toDatetimeLocalValue(urlTo) : ''
  );
  const [draftCompareFrom, setDraftCompareFrom] = useState('');
  const [draftCompareTo, setDraftCompareTo] = useState('');
  const [draftNotifyChannel, setDraftNotifyChannel] = useState<ExportHubNotifyChannel>('none');
  const [draftNotifyEmail, setDraftNotifyEmail] = useState('');
  const [draftNotifyWebhookUrl, setDraftNotifyWebhookUrl] = useState('');
  const [draftReportFormat, setDraftReportFormat] = useState<ReportFormat | ''>(() =>
    urlFormat === 'csv' || urlFormat === 'json' || urlFormat === 'xlsx' || urlFormat === 'zip'
      ? urlFormat
      : ''
  );
  const [draftDestination, setDraftDestination] = useState<ExportHubDestination>(() =>
    parseDestination(searchParams.get('destination'))
  );
  const [draftCampaignToggleCampaignId, setDraftCampaignToggleCampaignId] = useState('');
  const [draftCampaignToggleField, setDraftCampaignToggleField] = useState<CampaignToggleField | ''>(
    ''
  );
  const [draftCampaignToggleAt, setDraftCampaignToggleAt] = useState('');
  const [draftCampaignToggleWindowHours, setDraftCampaignToggleWindowHours] = useState('');
  const [draftLayerDesyncCount, setDraftLayerDesyncCount] = useState('');
  const [draftSpreadsheetId, setDraftSpreadsheetId] = useState('');
  const [draftSheetTitle, setDraftSheetTitle] = useState('');
  const [draftBillingFormat, setDraftBillingFormat] = useState<BillingFormat | ''>(() =>
    urlFormat === 'csv' || urlFormat === 'ndjson' ? urlFormat : ''
  );
  const [draftRedactPii, setDraftRedactPii] = useState(false);
  const [draftRowLimit, setDraftRowLimit] = useState(searchParams.get('row_limit') ?? '');
  const [draftJobId, setDraftJobId] = useState(jobId);
  const [recentJobs, setRecentJobs] = useState<ExportHubRecentJob[]>(() => listExportHubRecentJobs());
  const [jobStartedAtMs, setJobStartedAtMs] = useState<number | undefined>();
  const [activeRowLimit, setActiveRowLimit] = useState<number | undefined>();
  const completionToastRef = useRef<string | null>(null);
  const jobLoadErrorToastRef = useRef<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [rerunningJobId, setRerunningJobId] = useState<string | undefined>();
  const [downloading, setDownloading] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [pollToken, setPollToken] = useState(0);
  const [auditExportTruncated, setAuditExportTruncated] = useState(false);
  const [asyncStatusPhase, setAsyncStatusPhase] = useState<ExportHubStatusPhase>('idle');
  const [formValidationError, setFormValidationError] = useState<AdminValidationError | undefined>();
  const [presetName, setPresetName] = useState('');
  const [selectedViewId, setSelectedViewId] = useState('');
  const [savingPreset, setSavingPreset] = useState(false);
  const [exportingViewId, setExportingViewId] = useState<string | undefined>();
  const [savedViewsRefreshToken, setSavedViewsRefreshToken] = useState(0);

  const canManagePresets = sessionHasPermission(user?.permissions, 'campaigns:write');
  const canExportPreset = sessionHasPermission(user?.permissions, 'exports:run');
  const customerIdForViews = draftCustomerId.trim();
  const {
    data: savedViews,
    error: savedViewsError,
    fetching: savedViewsFetching,
  } = useResource(
    (signal) => {
      if (!customerIdForViews || !isUuidLike(customerIdForViews)) {
        return Promise.resolve(undefined);
      }
      return listSavedViews({ customer_id: customerIdForViews }, signal);
    },
    [customerIdForViews, savedViewsRefreshToken]
  );
  const refreshSavedViews = useCallback(() => {
    setSavedViewsRefreshToken((value) => value + 1);
  }, []);

  const reportFormatOptions = useMemo(
    () => resolveExportHubReportFormats(draftReportKey.trim() || 'placements', selectedCatalogRow),
    [draftReportKey, selectedCatalogRow]
  );

  const {
    data: googleSheetsStatus,
    error: googleSheetsStatusError,
    fetching: googleSheetsStatusFetching,
  } = useResource(
    (signal) => {
      if (selectedKind !== 'report' || draftDestination !== 'google_sheet') {
        return Promise.resolve(undefined);
      }
      return getGoogleSheetsStatus(signal);
    },
    [draftDestination, selectedKind]
  );

  const onRefreshCatalog = useCallback(() => {
    invalidateReportCatalogCache();
    setCatalogRefreshToken((value) => value + 1);
  }, []);

  const clearFormValidationError = useCallback(() => {
    setFormValidationError(undefined);
  }, []);

  const reportFormValidationFailure = useCallback((error: AdminValidationError) => {
    setFormValidationError(error);
    toastValidationError(error);
    setAsyncStatusPhase('idle');
  }, []);

  useEffect(() => {
    if (selectedEntry) {
      setCatalogPickerValue(exportHubCatalogOptionValue(selectedEntry));
      return;
    }
    const reportKey = searchParams.get('report_key')?.trim();
    if (reportKey) {
      setCatalogPickerValue(reportKey);
    }
  }, [searchParams, selectedEntry]);

  useEffect(() => {
    setDraftJobId(jobId);
  }, [jobId]);

  useEffect(() => {
    if (!selectedEntry) {
      return;
    }
    if (selectedEntry.reportKey) {
      setDraftReportKey(selectedEntry.reportKey);
    }
  }, [selectedEntry]);

  useEffect(() => {
    const customerId = searchParams.get('customer_id');
    if (customerId) {
      setDraftCustomerId(customerId);
    }
    const reportKey = searchParams.get('report_key');
    if (reportKey) {
      setDraftReportKey(reportKey);
    }
    const from = searchParams.get('from');
    if (from) {
      setDraftFrom(toDatetimeLocalValue(from));
    }
    const to = searchParams.get('to');
    if (to) {
      setDraftTo(toDatetimeLocalValue(to));
    }
    const format = searchParams.get('format');
    if (format === 'csv' || format === 'json' || format === 'xlsx' || format === 'zip') {
      setDraftReportFormat(format);
    }
    const destination = searchParams.get('destination');
    if (destination === 'download' || destination === 'google_sheet') {
      setDraftDestination(destination);
    }
    if (format === 'csv' || format === 'ndjson') {
      setDraftBillingFormat(format);
    }
    const rowLimit = searchParams.get('row_limit');
    if (rowLimit) {
      setDraftRowLimit(rowLimit);
    }
  }, [searchParams]);

  useEffect(() => {
    if (selectedKind !== 'report') {
      return;
    }
    if (draftReportFormat && reportFormatOptions.includes(draftReportFormat)) {
      return;
    }
    setDraftReportFormat(reportFormatOptions[0] ?? '');
  }, [draftReportFormat, reportFormatOptions, selectedKind]);

  useEffect(() => {
    if (draftDestination !== 'google_sheet') {
      return;
    }
    if (draftReportFormat === 'zip') {
      setDraftReportFormat(reportFormatOptions[0] ?? 'csv');
    }
  }, [draftDestination, draftReportFormat, reportFormatOptions]);

  const {
    data: reportJob,
    error: reportJobError,
    fetching: reportJobFetching,
    revalidating: reportJobRevalidating,
  } = useResource(
    (signal) => {
      if (!jobId || selectedKind !== 'report') {
        return Promise.resolve(undefined);
      }
      return getReportJob(jobId, signal);
    },
    [jobId, pollToken, selectedKind]
  );

  const {
    data: billingJob,
    error: billingJobError,
    fetching: billingJobFetching,
    revalidating: billingJobRevalidating,
  } = useResource(
    (signal) => {
      if (!jobId || selectedKind !== 'billing') {
        return Promise.resolve(undefined);
      }
      return getBillingExportJob(jobId, signal);
    },
    [jobId, pollToken, selectedKind]
  );

  const job = selectedKind === 'billing' ? billingJob : reportJob;
  const jobError = selectedKind === 'billing' ? billingJobError : reportJobError;
  const jobErrorMessage = useMemo(
    () => exportHubJobErrorMessage(job && 'error' in job ? job.error : undefined),
    [job]
  );

  const jobFetching = selectedKind === 'billing' ? billingJobFetching : reportJobFetching;
  const jobRevalidating = selectedKind === 'billing' ? billingJobRevalidating : reportJobRevalidating;
  const jobPending = Boolean(job?.status && isJobPendingStatus(String(job.status)));
  const autoPolling =
    Boolean(jobId) && selectedKind !== 'audit' && selectedKind !== 'directory' && jobPending;
  const polling = Boolean(jobId) && (jobFetching || jobRevalidating || autoPolling);

  useEffect(() => {
    if (!autoPolling) {
      return;
    }
    const timer = window.setInterval(() => {
      setPollToken((value) => value + 1);
    }, EXPORT_HUB_POLL_INTERVAL_MS);
    return () => {
      window.clearInterval(timer);
    };
  }, [autoPolling, jobId]);

  useEffect(() => {
    if (!jobId || !job?.status) {
      return;
    }
    const status = String(job.status);
    const patch = {
      status,
      bytes: job.bytes ?? undefined,
      error: job.error ?? jobErrorMessage,
      spreadsheetUrl:
        selectedKind === 'report' ? exportJobSpreadsheetUrl(reportJob) : undefined,
    };
    setRecentJobs(patchExportHubRecentJob(jobId, patch));

    const toastKey = `${jobId}:${normalizeExportJobStatus(status)}`;
    if (completionToastRef.current === toastKey) {
      return;
    }
    if (isJobReadyStatus(status)) {
      completionToastRef.current = toastKey;
      const spreadsheetUrl =
        selectedKind === 'report' ? exportJobSpreadsheetUrl(reportJob) : undefined;
      if (spreadsheetUrl) {
        toast.success('Export ready in Google Sheets');
      } else {
        toast.success('Export ready for download');
      }
      return;
    }
    if (exportJobPhase(status) === 'failed') {
      completionToastRef.current = toastKey;
      toast.error(jobErrorMessage ?? 'Export failed');
    }
  }, [job, jobErrorMessage, jobId, reportJob, selectedKind]);

  const rowLimit = useMemo(
    () => clampExportHubRowLimit(parseExportHubRowLimitDraft(draftRowLimit, rowLimitBounds), rowLimitBounds),
    [draftRowLimit, rowLimitBounds]
  );

  useEffect(() => {
    if (!jobId || !jobError) {
      return;
    }
    const message = exportHubErrorMessage(jobError, 'Job load failed');
    const toastKey = `${jobId}:load:${message}`;
    if (jobLoadErrorToastRef.current === toastKey) {
      return;
    }
    jobLoadErrorToastRef.current = toastKey;
    toast.error(message);
  }, [jobError, jobId]);

  useEffect(() => {
    if (!jobPending && !creating) {
      return;
    }
    setJobStartedAtMs((current) => current ?? Date.now());
  }, [creating, jobId, jobPending]);

  useEffect(() => {
    if (!jobId) {
      setJobStartedAtMs(undefined);
      setActiveRowLimit(undefined);
    }
  }, [jobId]);

  useEffect(() => {
    if (creating || downloading || cancelling || polling) {
      setAsyncStatusPhase('pending');
      return;
    }
    const status = (job?.status ?? '').toString();
    if (isJobPendingStatus(status)) {
      setAsyncStatusPhase('pending');
      return;
    }
    setAsyncStatusPhase('idle');
  }, [creating, downloading, cancelling, job?.status, polling]);

  useEffect(() => {
    if (asyncStatusPhase !== 'pending') {
      toast.dismiss(EXPORT_PENDING_TOAST_ID);
      return;
    }
    toast.loading(resolveExportPendingToastMessage(creating, downloading, cancelling), {
      id: EXPORT_PENDING_TOAST_ID,
      duration: Infinity,
    });
  }, [asyncStatusPhase, cancelling, creating, downloading]);

  const recordRecentJob = useCallback(
    (
      nextJobId: string,
      rowLimitValue: number,
      status = 'pending',
      meta?: Pick<ExportHubRecentJob, 'format' | 'destination' | 'spreadsheetUrl'>
    ) => {
      const label = recentJobLabel(selectedKind, selectedEntry, draftReportKey.trim());
      setRecentJobs(
        upsertExportHubRecentJob({
          jobId: nextJobId,
          kind: selectedKind,
          label,
          customerId: draftCustomerId.trim() || undefined,
          rowLimit: rowLimitValue,
          format: meta?.format,
          destination: meta?.destination,
          spreadsheetUrl: meta?.spreadsheetUrl,
          status,
          createdAt: new Date().toISOString(),
        })
      );
      setJobStartedAtMs(Date.now());
      setActiveRowLimit(rowLimitValue);
      completionToastRef.current = null;
    },
    [draftCustomerId, draftReportKey, selectedEntry, selectedKind]
  );

  const onCatalogPickerChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setCatalogPickerValue(value);
      const resolved = resolveExportHubCatalogValue(value, catalogEntries);
      const next = new URLSearchParams(searchParams);
      if (resolved.entry) {
        next.set('entry', resolved.entry.id);
        next.set('kind', resolved.entry.kind);
        if (resolved.entry.reportKey) {
          next.set('report_key', resolved.entry.reportKey);
        } else {
          next.delete('report_key');
        }
      } else if (resolved.customReportKey) {
        next.delete('entry');
        next.set('kind', 'report');
        next.set('report_key', resolved.customReportKey);
      }
      replaceSearchParams(next);
    },
    [catalogEntries, clearFormValidationError, replaceSearchParams, searchParams]
  );

  const notifyRowLimitWarning = useCallback((message: string) => {
    toast.warning(message, { duration: 3000 });
  }, []);

  const notifyRowLimitError = useCallback((message: string) => {
    toast.error(message, { duration: 3000 });
  }, []);

  const onDraftRowLimitChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftRowLimit(value);
    },
    [clearFormValidationError]
  );

  const onDraftCustomerIdChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCustomerId(value);
    },
    [clearFormValidationError]
  );

  const onDraftReportKeyChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftReportKey(value);
    },
    [clearFormValidationError]
  );

  const onDraftFromChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftFrom(value);
    },
    [clearFormValidationError]
  );

  const onDraftToChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftTo(value);
    },
    [clearFormValidationError]
  );

  const onDraftCompareFromChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCompareFrom(value);
    },
    [clearFormValidationError]
  );

  const onDraftCompareToChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCompareTo(value);
    },
    [clearFormValidationError]
  );

  const onDraftNotifyChannelChange = useCallback(
    (value: ExportHubNotifyChannel) => {
      clearFormValidationError();
      setDraftNotifyChannel(value);
    },
    [clearFormValidationError]
  );

  const onDraftNotifyEmailChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftNotifyEmail(value);
    },
    [clearFormValidationError]
  );

  const onDraftNotifyWebhookUrlChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftNotifyWebhookUrl(value);
    },
    [clearFormValidationError]
  );

  const onDraftReportFormatChange = useCallback(
    (value: ReportFormat) => {
      clearFormValidationError();
      setDraftReportFormat(value);
    },
    [clearFormValidationError]
  );

  const onDraftDestinationChange = useCallback(
    (value: ExportHubDestination) => {
      clearFormValidationError();
      setDraftDestination(value);
      const next = new URLSearchParams(searchParams);
      if (value === 'download') {
        next.delete('destination');
      } else {
        next.set('destination', value);
      }
      replaceSearchParams(next);
    },
    [clearFormValidationError, replaceSearchParams, searchParams]
  );

  const onDraftCampaignToggleCampaignIdChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCampaignToggleCampaignId(value);
    },
    [clearFormValidationError]
  );

  const onDraftCampaignToggleFieldChange = useCallback(
    (value: CampaignToggleField) => {
      clearFormValidationError();
      setDraftCampaignToggleField(value);
    },
    [clearFormValidationError]
  );

  const onDraftCampaignToggleAtChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCampaignToggleAt(value);
    },
    [clearFormValidationError]
  );

  const onDraftCampaignToggleWindowHoursChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCampaignToggleWindowHours(value);
    },
    [clearFormValidationError]
  );

  const onDraftLayerDesyncCountChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftLayerDesyncCount(value);
    },
    [clearFormValidationError]
  );

  const onDraftSpreadsheetIdChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftSpreadsheetId(value);
    },
    [clearFormValidationError]
  );

  const onDraftSheetTitleChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftSheetTitle(value);
    },
    [clearFormValidationError]
  );

  const onDraftBillingFormatChange = useCallback(
    (value: BillingFormat) => {
      clearFormValidationError();
      setDraftBillingFormat(value);
    },
    [clearFormValidationError]
  );

  const onDraftRowLimitBlur = useCallback(() => {
    if (!draftRowLimit.trim()) {
      return;
    }
    const normalized = normalizeExportHubRowLimitDraft(draftRowLimit, rowLimitBounds);
    setDraftRowLimit(String(normalized.value));
    if (normalized.wasClamped) {
      notifyRowLimitWarning(
        `Adjusted to tier maximum of ${normalized.value.toLocaleString()} rows.`
      );
      return;
    }
    if (normalized.wasInvalid) {
      notifyRowLimitError('Invalid row limit');
      setDraftRowLimit('');
    }
  }, [draftRowLimit, notifyRowLimitError, notifyRowLimitWarning, rowLimitBounds]);

  const buildReportImportPayload = useCallback((): ValidationResult<Record<string, unknown>> => {
    const reportKey = draftReportKey.trim();
    if (reportKey === 'campaign-toggle-cohort') {
      const campaignIdCheck = requireNonEmpty(
        draftCampaignToggleCampaignId,
        'Campaign ID',
        'campaign_toggle_campaign_id'
      );
      if (!campaignIdCheck.ok) {
        return campaignIdCheck;
      }
      if (!isUuidLike(campaignIdCheck.value)) {
        return {
          ok: false,
          error: validationError('Campaign ID must be a valid UUID.', {
            field: 'campaign_toggle_campaign_id',
          }),
        };
      }
      const toggleFieldCheck = requireNonEmpty(
        draftCampaignToggleField,
        'Toggle field',
        'campaign_toggle_field'
      );
      if (!toggleFieldCheck.ok) {
        return toggleFieldCheck;
      }
      let windowHours = 72;
      if (draftCampaignToggleWindowHours.trim()) {
        const windowHoursCheck = requireInteger(draftCampaignToggleWindowHours, 'Window hours', {
          min: 1,
          max: 168,
          field: 'campaign_toggle_window_hours',
        });
        if (!windowHoursCheck.ok) {
          return windowHoursCheck;
        }
        windowHours = windowHoursCheck.value;
      }
      const payload: Record<string, unknown> = {
        campaign_id: campaignIdCheck.value,
        toggle_field: toggleFieldCheck.value,
        window_hours: windowHours,
      };
      if (draftCampaignToggleAt.trim()) {
        const toggleAtIso = fromDatetimeLocalValue(draftCampaignToggleAt);
        if (!toggleAtIso) {
          return {
            ok: false,
            error: validationError('Toggle time must be a valid datetime.', {
              field: 'campaign_toggle_at',
            }),
          };
        }
        payload.toggle_at = toggleAtIso;
      }
      return { ok: true, value: payload };
    }
    if (reportKey === 'layer-desync-drilldown') {
      if (!draftLayerDesyncCount.trim()) {
        return { ok: true, value: { layer_desync_count: 2 } };
      }
      const countCheck = requireInteger(draftLayerDesyncCount, 'Layer desync count', {
        min: 1,
        max: 255,
        field: 'layer_desync_count',
      });
      if (!countCheck.ok) {
        return countCheck;
      }
      return { ok: true, value: { layer_desync_count: countCheck.value } };
    }
    return { ok: true, value: {} };
  }, [
    draftCampaignToggleAt,
    draftCampaignToggleCampaignId,
    draftCampaignToggleField,
    draftCampaignToggleWindowHours,
    draftLayerDesyncCount,
    draftReportKey,
  ]);

  const onRunExport = useCallback(async () => {
    setAsyncStatusPhase('pending');
    if (selectedKind === 'directory') {
      setAsyncStatusPhase('idle');
      return;
    }
    if (selectedKind === 'audit') {
      setCreating(true);
      try {
        const result = await exportAuditCsv({
          format: 'csv',
          redact_pii: draftRedactPii || undefined,
        });
        triggerBlobDownload(result.blob, 'audit-export.csv');
        setAuditExportTruncated(result.truncated);
        setAsyncStatusPhase('idle');
        toast.success('Audit CSV exported');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
        setAsyncStatusPhase('idle');
      } finally {
        setCreating(false);
      }
      return;
    }

    const customerIdCheck = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerIdCheck.ok) {
      reportFormValidationFailure(customerIdCheck.error);
      return;
    }
    const customerId = customerIdCheck.value;

    const dateRangeCheck = requireDateRange(draftFrom, draftTo);
    if (!dateRangeCheck.ok) {
      reportFormValidationFailure(dateRangeCheck.error);
      return;
    }
    const fromIso = fromDatetimeLocalValue(dateRangeCheck.value.from);
    const toIso = fromDatetimeLocalValue(dateRangeCheck.value.to);
    if (!fromIso || !toIso) {
      reportFormValidationFailure(
        validationError('From date and To date must be valid.', { field: 'from' })
      );
      return;
    }

    const hasCompareFrom = draftCompareFrom.trim() !== '';
    const hasCompareTo = draftCompareTo.trim() !== '';
    if (hasCompareFrom !== hasCompareTo) {
      reportFormValidationFailure(
        validationError('Compare from and Compare to must both be set.', { field: 'compare_from' })
      );
      return;
    }
    let compareFromIso: string | undefined;
    let compareToIso: string | undefined;
    if (hasCompareFrom && hasCompareTo) {
      const compareRangeCheck = requireDateRange(draftCompareFrom, draftCompareTo);
      if (!compareRangeCheck.ok) {
        reportFormValidationFailure(compareRangeCheck.error);
        return;
      }
      compareFromIso = fromDatetimeLocalValue(compareRangeCheck.value.from);
      compareToIso = fromDatetimeLocalValue(compareRangeCheck.value.to);
      if (!compareFromIso || !compareToIso) {
        reportFormValidationFailure(
          validationError('Compare dates must be valid.', { field: 'compare_from' })
        );
        return;
      }
    }

    if (selectedKind === 'report' && draftNotifyChannel === 'email' && !draftNotifyEmail.trim()) {
      reportFormValidationFailure(
        validationError('Notify email is required for email channel.', { field: 'notify_email' })
      );
      return;
    }
    if (
      selectedKind === 'report' &&
      draftNotifyChannel === 'slack_webhook' &&
      !draftNotifyWebhookUrl.trim()
    ) {
      reportFormValidationFailure(
        validationError('Slack webhook URL is required for Slack channel.', {
          field: 'notify_webhook_url',
        })
      );
      return;
    }

    const rowLimitCheck = requireNonEmpty(draftRowLimit, 'Row limit', 'row_limit');
    if (!rowLimitCheck.ok) {
      reportFormValidationFailure(rowLimitCheck.error);
      return;
    }
    const rowLimitParsed = requireInteger(rowLimitCheck.value, 'Row limit', {
      min: rowLimitBounds.min,
      max: rowLimitBounds.max,
      field: 'row_limit',
    });
    if (!rowLimitParsed.ok) {
      reportFormValidationFailure(rowLimitParsed.error);
      return;
    }
    const effectiveRowLimit = rowLimitParsed.value;
    setDraftRowLimit(String(effectiveRowLimit));

    if (selectedKind === 'report') {
      const formatCheck = requireNonEmpty(draftReportFormat, 'Format', 'format');
      if (!formatCheck.ok) {
        reportFormValidationFailure(formatCheck.error);
        return;
      }
    }
    if (selectedKind === 'billing') {
      const formatCheck = requireNonEmpty(draftBillingFormat, 'Format', 'format');
      if (!formatCheck.ok) {
        reportFormValidationFailure(formatCheck.error);
        return;
      }
    }

    clearFormValidationError();
    setCreating(true);
    try {
      if (selectedKind === 'billing') {
        const created = await createBillingExportJob({
          customer_id: customerId,
          from: fromIso,
          to: toIso,
          format: draftBillingFormat as BillingFormat,
          row_limit: effectiveRowLimit,
        });
        const nextId = created.job_id;
        if (!nextId) {
          throw validationError('job_id missing in create response', { kind: 'action' });
        }
        const next = new URLSearchParams(searchParams);
        next.set('job_id', nextId);
        next.set('customer_id', customerId);
        next.set('kind', 'billing');
        next.set('row_limit', String(effectiveRowLimit));
        next.set('from', fromIso);
        next.set('to', toIso);
        if (selectedEntry?.id) {
          next.set('entry', selectedEntry.id);
        }
        replaceSearchParams(next);
        setDraftJobId(nextId);
        recordRecentJob(nextId, effectiveRowLimit);
        setPollToken((value) => value + 1);
        toast.success('Billing export job enqueued');
        return;
      }

      const reportKeyCheck = requireNonEmpty(draftReportKey, 'Report key', 'report_key');
      if (!reportKeyCheck.ok) {
        reportFormValidationFailure(reportKeyCheck.error);
        return;
      }
      const reportKey = reportKeyCheck.value;
      if (draftDestination === 'google_sheet') {
        if (googleSheetsStatusError) {
          reportFormValidationFailure(
            validationError(exportHubErrorMessage(googleSheetsStatusError, 'Google Sheets status unavailable'), {
              kind: 'action',
            })
          );
          return;
        }
        if (!googleSheetsStatus?.connected) {
          reportFormValidationFailure(
            validationError('Connect Google Sheets in Integrations before exporting to a spreadsheet.', {
              kind: 'action',
            })
          );
          return;
        }
      }
      const importPayloadCheck = buildReportImportPayload();
      if (!importPayloadCheck.ok) {
        reportFormValidationFailure(importPayloadCheck.error);
        return;
      }
      const importPayload =
        reportKey === 'campaign-toggle-cohort' || reportKey === 'layer-desync-drilldown'
          ? importPayloadCheck.value
          : Object.keys(importPayloadCheck.value).length > 0
            ? importPayloadCheck.value
            : undefined;
      const spreadsheetId = draftSpreadsheetId.trim();
      const sheetTitle = draftSheetTitle.trim();
      const created = await createReportJob({
        customer_id: customerId,
        report_key: reportKey,
        from: fromIso,
        to: toIso,
        compare_from: compareFromIso,
        compare_to: compareToIso,
        format: draftReportFormat as ReportFormat,
        row_limit: effectiveRowLimit,
        destination: draftDestination,
        import_payload: importPayload,
        notify:
          draftNotifyChannel === 'none'
            ? undefined
            : {
                channel: draftNotifyChannel,
                email: draftNotifyEmail.trim() || undefined,
                webhook_url: draftNotifyWebhookUrl.trim() || undefined,
              },
        google_sheet:
          draftDestination === 'google_sheet'
            ? {
                mode: spreadsheetId ? 'append' : 'create',
                spreadsheet_id: spreadsheetId || undefined,
                sheet_title: sheetTitle || undefined,
              }
            : undefined,
      });
      const nextId = created.id ?? created.job_id;
      if (!nextId) {
        throw validationError('job_id missing in create response', { kind: 'action' });
      }
      const next = new URLSearchParams(searchParams);
      next.set('job_id', nextId);
      next.set('customer_id', customerId);
      next.set('report_key', reportKey);
      next.set('kind', 'report');
      next.set('row_limit', String(effectiveRowLimit));
      next.set('from', fromIso);
      next.set('to', toIso);
      if (draftDestination === 'google_sheet') {
        next.set('destination', 'google_sheet');
      } else {
        next.delete('destination');
      }
      if (selectedEntry?.id && !selectedEntry.id.startsWith('custom-')) {
        next.set('entry', selectedEntry.id);
      }
      replaceSearchParams(next);
      setDraftJobId(nextId);
      recordRecentJob(nextId, effectiveRowLimit, 'pending', {
        format: draftReportFormat as ReportFormat,
        destination: draftDestination,
      });
      setPollToken((value) => value + 1);
      toast.success('Report export job enqueued');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
      setAsyncStatusPhase('idle');
    } finally {
      setCreating(false);
    }
  }, [
    buildReportImportPayload,
    draftBillingFormat,
    draftCampaignToggleAt,
    draftCampaignToggleCampaignId,
    draftCampaignToggleField,
    draftCampaignToggleWindowHours,
    draftCompareFrom,
    draftCompareTo,
    draftCustomerId,
    draftDestination,
    draftFrom,
    draftLayerDesyncCount,
    draftNotifyChannel,
    draftNotifyEmail,
    draftNotifyWebhookUrl,
    draftRedactPii,
    draftReportFormat,
    draftReportKey,
    draftSheetTitle,
    draftSpreadsheetId,
    draftTo,
    googleSheetsStatus?.connected,
    googleSheetsStatusError,
    replaceSearchParams,
    recordRecentJob,
    clearFormValidationError,
    reportFormValidationFailure,
    rowLimitBounds,
    searchParams,
    selectedEntry?.id,
    selectedKind,
  ]);

  const onSelectRecentJob = useCallback(
    (targetJobId: string) => {
      const recent = recentJobs.find((row) => row.jobId === targetJobId);
      const next = new URLSearchParams(searchParams);
      next.set('job_id', targetJobId);
      if (recent) {
        next.set('kind', recent.kind);
        if (recent.customerId) {
          next.set('customer_id', recent.customerId);
        }
        if (recent.rowLimit != null) {
          next.set('row_limit', String(recent.rowLimit));
          setActiveRowLimit(recent.rowLimit);
        }
      }
      replaceSearchParams(next);
      setDraftJobId(targetJobId);
      setPollToken((value) => value + 1);
    },
    [recentJobs, replaceSearchParams, searchParams]
  );

  const onRerunRecentJob = useCallback(
    async (recent: ExportHubRecentJob) => {
      if (recent.kind !== 'report') {
        return;
      }
      setRerunningJobId(recent.jobId);
      setAsyncStatusPhase('pending');
      try {
        const created = await rerunReportJob(recent.jobId);
        const nextId = created.id ?? created.job_id;
        if (!nextId) {
          throw validationError('job_id missing in rerun response', { kind: 'action' });
        }
        const next = new URLSearchParams(searchParams);
        next.set('job_id', nextId);
        next.set('kind', 'report');
        if (recent.customerId) {
          next.set('customer_id', recent.customerId);
        }
        replaceSearchParams(next);
        setDraftJobId(nextId);
        recordRecentJob(nextId, recent.rowLimit ?? rowLimit, 'pending', {
          format: recent.format,
          destination: recent.destination,
        });
        setPollToken((value) => value + 1);
        toast.success('Export job re-run enqueued');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
        setAsyncStatusPhase('idle');
      } finally {
        setRerunningJobId(undefined);
      }
    },
    [recordRecentJob, replaceSearchParams, rowLimit, searchParams]
  );

  const onRetryRecentJob = useCallback(
    (recent: ExportHubRecentJob) => {
      flushSync(() => {
        if (recent.customerId) {
          setDraftCustomerId(recent.customerId);
        }
        if (recent.rowLimit != null) {
          setDraftRowLimit(String(recent.rowLimit));
        }
      });
      void onRunExport();
    },
    [onRunExport]
  );

  const onDownloadRecentJob = useCallback(
    async (recent: ExportHubRecentJob) => {
      setDownloading(true);
      try {
        if (recent.kind === 'billing') {
          const blob = await downloadBillingExportJob(recent.jobId);
          triggerBlobDownload(blob, `billing-export-${recent.jobId}.csv`);
        } else {
          const blob = await downloadReportJob(recent.jobId);
          const ext = reportDownloadExtension(recent.format);
          triggerBlobDownload(blob, `${draftReportKey || 'report'}.${ext}`);
        }
        toast.success('Export downloaded');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
      } finally {
        setDownloading(false);
      }
    },
    [draftReportKey]
  );

  const onPollJob = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftJobId.trim();
    if (trimmed) {
      next.set('job_id', trimmed);
    } else {
      next.delete('job_id');
    }
    replaceSearchParams(next);
    setPollToken((value) => value + 1);
  }, [draftJobId, replaceSearchParams, searchParams]);

  const onCancelJob = useCallback(async () => {
    const trimmed = draftJobId.trim();
    if (!trimmed || selectedKind !== 'report') {
      return;
    }
    if (!confirmDestructiveAction(`Cancel export job ${trimmed}?`)) {
      return;
    }
    setCancelling(true);
    setAsyncStatusPhase('pending');
    try {
      await cancelReportJob(trimmed);
      setPollToken((value) => value + 1);
      toast.success('Export job cancelled');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
      setAsyncStatusPhase('idle');
    } finally {
      setCancelling(false);
    }
  }, [draftJobId, selectedKind]);

  const onDownloadJob = useCallback(async () => {
    const trimmed = draftJobId.trim();
    if (!trimmed) {
      return;
    }
    setDownloading(true);
    setAsyncStatusPhase('pending');
    try {
      if (selectedKind === 'billing') {
        const blob = await downloadBillingExportJob(trimmed);
        const ext = draftBillingFormat === 'ndjson' ? 'ndjson' : 'csv';
        triggerBlobDownload(blob, `billing-export-${trimmed}.${ext}`);
      } else {
        const blob = await downloadReportJob(trimmed);
        const ext = reportDownloadExtension(
          (reportJob?.format ?? draftReportFormat) || undefined
        );
        triggerBlobDownload(blob, `${draftReportKey || 'report'}.${ext}`);
      }
      setAsyncStatusPhase('idle');
      toast.success('Export downloaded');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
      setAsyncStatusPhase('idle');
    } finally {
      setDownloading(false);
    }
  }, [draftBillingFormat, draftJobId, draftReportFormat, draftReportKey, reportJob?.format, selectedKind]);

  const applySavedViewPatch = useCallback(
    (patch: ExportHubSavedViewApplyPatch, viewReportKey: string) => {
      flushSync(() => {
        if (patch.from !== undefined) {
          setDraftFrom(patch.from);
        }
        if (patch.to !== undefined) {
          setDraftTo(patch.to);
        }
        if (patch.compareFrom !== undefined) {
          setDraftCompareFrom(patch.compareFrom);
        }
        if (patch.compareTo !== undefined) {
          setDraftCompareTo(patch.compareTo);
        }
        if (patch.notifyChannel !== undefined) {
          setDraftNotifyChannel(patch.notifyChannel);
        }
        if (patch.notifyEmail !== undefined) {
          setDraftNotifyEmail(patch.notifyEmail);
        }
        if (patch.notifyWebhookUrl !== undefined) {
          setDraftNotifyWebhookUrl(patch.notifyWebhookUrl);
        }
        if (patch.reportFormat !== undefined) {
          setDraftReportFormat(patch.reportFormat);
        }
        if (patch.destination !== undefined) {
          setDraftDestination(patch.destination);
        }
        if (patch.rowLimit !== undefined) {
          setDraftRowLimit(patch.rowLimit);
        }
        if (patch.billingFormat !== undefined) {
          setDraftBillingFormat(patch.billingFormat);
        }
        if (patch.redactPii !== undefined) {
          setDraftRedactPii(patch.redactPii);
        }
        if (patch.spreadsheetId !== undefined) {
          setDraftSpreadsheetId(patch.spreadsheetId);
        }
        if (patch.sheetTitle !== undefined) {
          setDraftSheetTitle(patch.sheetTitle);
        }
        if (patch.campaignToggleCampaignId !== undefined) {
          setDraftCampaignToggleCampaignId(patch.campaignToggleCampaignId);
        }
        if (patch.campaignToggleField !== undefined) {
          setDraftCampaignToggleField(patch.campaignToggleField);
        }
        if (patch.campaignToggleAt !== undefined) {
          setDraftCampaignToggleAt(patch.campaignToggleAt);
        }
        if (patch.campaignToggleWindowHours !== undefined) {
          setDraftCampaignToggleWindowHours(patch.campaignToggleWindowHours);
        }
        if (patch.layerDesyncCount !== undefined) {
          setDraftLayerDesyncCount(patch.layerDesyncCount);
        }
        if (patch.reportKey) {
          setDraftReportKey(patch.reportKey);
        }
      });

      const next = new URLSearchParams(searchParams);
      const kind = patch.kind ?? 'report';
      next.set('kind', kind);
      if (patch.entryId) {
        next.set('entry', patch.entryId);
        const entry = findExportHubEntry(patch.entryId, catalog?.rows);
        if (entry?.reportKey) {
          next.set('report_key', entry.reportKey);
          setCatalogPickerValue(exportHubCatalogOptionValue(entry));
        }
      } else if (patch.reportKey || viewReportKey) {
        const reportKey = patch.reportKey ?? viewReportKey;
        next.delete('entry');
        next.set('report_key', reportKey);
        setCatalogPickerValue(reportKey);
      }
      if (patch.destination === 'google_sheet') {
        next.set('destination', 'google_sheet');
      } else if (patch.destination === 'download') {
        next.delete('destination');
      }
      if (patch.reportFormat) {
        next.set('format', patch.reportFormat);
      }
      replaceSearchParams(next);
      clearFormValidationError();
    },
    [catalog?.rows, clearFormValidationError, replaceSearchParams, searchParams]
  );

  const onLoadSavedView = useCallback(() => {
    const view = savedViews?.find((row) => row.id === selectedViewId);
    if (!view) {
      toast.error('Select a saved preset to load');
      return;
    }
    const patch = applyExportHubSavedViewSpec(view.report_key ?? 'placements', view.spec);
    applySavedViewPatch(patch, view.report_key ?? 'placements');
    toast.success('Saved preset loaded');
  }, [applySavedViewPatch, savedViews, selectedViewId]);

  const onSaveSavedView = useCallback(async () => {
    if (!canManagePresets) {
      toast.error('campaigns:write permission required');
      return;
    }
    const customerIdCheck = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerIdCheck.ok) {
      reportFormValidationFailure(customerIdCheck.error);
      return;
    }
    const nameCheck = requireNonEmpty(presetName, 'Preset name', 'preset_name');
    if (!nameCheck.ok) {
      reportFormValidationFailure(nameCheck.error);
      return;
    }
    const spec = buildExportHubSavedViewSpec({
      selectedKind,
      entryId: selectedEntry?.id,
      reportKey: draftReportKey,
      from: draftFrom,
      to: draftTo,
      compareFrom: draftCompareFrom,
      compareTo: draftCompareTo,
      notifyChannel: draftNotifyChannel,
      notifyEmail: draftNotifyEmail,
      notifyWebhookUrl: draftNotifyWebhookUrl,
      reportFormat: draftReportFormat,
      destination: draftDestination,
      rowLimit: draftRowLimit,
      billingFormat: draftBillingFormat,
      redactPii: draftRedactPii,
      spreadsheetId: draftSpreadsheetId,
      sheetTitle: draftSheetTitle,
      campaignToggleCampaignId: draftCampaignToggleCampaignId,
      campaignToggleField: draftCampaignToggleField,
      campaignToggleAt: draftCampaignToggleAt,
      campaignToggleWindowHours: draftCampaignToggleWindowHours,
      layerDesyncCount: draftLayerDesyncCount,
    });
    const reportKey = resolveSavedViewReportKey(selectedKind, draftReportKey);
    setSavingPreset(true);
    try {
      if (selectedViewId) {
        await updateSavedView(selectedViewId, {
          name: nameCheck.value,
          report_key: reportKey,
          spec,
          is_shared: false,
        });
        toast.success('Saved preset updated');
      } else {
        const created = await createSavedView({
          customer_id: customerIdCheck.value,
          name: nameCheck.value,
          report_key: reportKey,
          spec,
          is_shared: false,
        });
        if (created.id) {
          setSelectedViewId(created.id);
        }
        toast.success('Saved preset created');
      }
      refreshSavedViews();
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
    } finally {
      setSavingPreset(false);
    }
  }, [
    canManagePresets,
    draftBillingFormat,
    draftCampaignToggleAt,
    draftCampaignToggleCampaignId,
    draftCampaignToggleField,
    draftCampaignToggleWindowHours,
    draftCompareFrom,
    draftCompareTo,
    draftCustomerId,
    draftDestination,
    draftFrom,
    draftLayerDesyncCount,
    draftNotifyChannel,
    draftNotifyEmail,
    draftNotifyWebhookUrl,
    draftRedactPii,
    draftReportFormat,
    draftReportKey,
    draftRowLimit,
    draftSheetTitle,
    draftSpreadsheetId,
    draftTo,
    presetName,
    refreshSavedViews,
    reportFormValidationFailure,
    selectedEntry?.id,
    selectedKind,
    selectedViewId,
  ]);

  const onDeleteSavedView = useCallback(async () => {
    if (!canManagePresets || !selectedViewId) {
      return;
    }
    const view = savedViews?.find((row) => row.id === selectedViewId);
    if (!view) {
      return;
    }
    if (!confirmDestructiveAction(`Delete saved preset "${view.name}"?`)) {
      return;
    }
    setSavingPreset(true);
    try {
      await deleteSavedView(selectedViewId);
      setSelectedViewId('');
      setPresetName('');
      refreshSavedViews();
      toast.success('Saved preset deleted');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
    } finally {
      setSavingPreset(false);
    }
  }, [canManagePresets, refreshSavedViews, savedViews, selectedViewId]);

  const onExportSavedView = useCallback(async () => {
    if (!canExportPreset || !selectedViewId) {
      toast.error('exports:run permission required');
      return;
    }
    setExportingViewId(selectedViewId);
    setAsyncStatusPhase('pending');
    try {
      const created = await exportSavedView(selectedViewId);
      const nextId = created.job_id;
      if (!nextId) {
        throw validationError('job_id missing in export response', { kind: 'action' });
      }
      const view = created.view ?? savedViews?.find((row) => row.id === selectedViewId);
      const reportKey = view?.report_key ?? draftReportKey;
      const next = new URLSearchParams(searchParams);
      next.set('job_id', nextId);
      next.set('customer_id', draftCustomerId.trim());
      next.set('report_key', reportKey);
      next.set('kind', 'report');
      replaceSearchParams(next);
      setDraftJobId(nextId);
      recordRecentJob(nextId, rowLimit, 'pending', {
        format: draftReportFormat || 'csv',
        destination: draftDestination,
      });
      setPollToken((value) => value + 1);
      toast.success('Export job enqueued from saved preset');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
      setAsyncStatusPhase('idle');
    } finally {
      setExportingViewId(undefined);
    }
  }, [
    canExportPreset,
    draftCustomerId,
    draftDestination,
    draftReportFormat,
    draftReportKey,
    recordRecentJob,
    replaceSearchParams,
    rowLimit,
    savedViews,
    searchParams,
    selectedViewId,
  ]);

  const directoryLinkHref =
    selectedKind === 'directory' ? selectedEntry?.href?.trim() || '/campaigns' : undefined;

  return {
    catalogEntries,
    catalogError,
    catalogFetching,
    catalogHasSnapshot,
    onRefreshCatalog,
    catalogPickerValue,
    selectedKind,
    directoryLinkHref,
    selectedEntryDescription: selectedEntry?.description,
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
    draftBillingFormat,
    draftRedactPii,
    draftRowLimit,
    rowLimitBounds,
    rowLimit,
    draftJobId,
    job,
    jobSpreadsheetUrl: selectedKind === 'report' ? exportJobSpreadsheetUrl(reportJob) : undefined,
    jobPending,
    autoPolling,
    jobStartedAtMs,
    activeRowLimit,
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
    onDraftReportKeyChange,
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
    onDraftRedactPiiChange: setDraftRedactPii,
    onDraftRowLimitChange,
    onDraftRowLimitBlur,
    onDraftJobIdChange: setDraftJobId,
    onRunExport,
    onPollJob,
    onCancelJob,
    onDownloadJob,
    onSelectRecentJob,
    onDownloadRecentJob,
    onRerunRecentJob,
    onRetryRecentJob,
    savedViews: savedViews ?? [],
    savedViewsHasSnapshot: savedViews != null,
    savedViewsError,
    savedViewsFetching,
    canManagePresets,
    canExportPreset,
    presetName,
    selectedViewId,
    savingPreset,
    exportingViewId,
    onPresetNameChange: setPresetName,
    onSelectedViewIdChange: setSelectedViewId,
    onLoadSavedView,
    onSaveSavedView,
    onDeleteSavedView,
    onExportSavedView,
  };
}
