import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import { toast } from 'sonner';

import { exportAuditCsv } from '@/api/audit_api';
import {
  createBillingExportJob,
  downloadBillingExportJob,
  getBillingExportJob,
} from '@/api/billing_api';
import {
  cancelReportJob,
  createReportJob,
  downloadReportJob,
  getReportJob,
} from '@/api/reports_api';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import {
  exportHubCatalogOptionValue,
  exportHubEntriesFromCatalog,
  findExportHubEntry,
  resolveExportHubCatalogValue,
  type ExportHubEntry,
  type ExportHubKind,
} from '@/domains/exports/export_hub_catalog';
import {
  clampExportHubRowLimit,
  normalizeExportHubRowLimitDraft,
  parseExportHubRowLimitDraft,
  resolveExportHubRowLimitBounds,
} from '@/domains/exports/export_hub_limits';
import { exportHubErrorMessage, exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';
import {
  exportJobPhase,
  normalizeExportJobStatus,
} from '@/domains/exports/export_hub_job_status';
import {
  listExportHubRecentJobs,
  patchExportHubRecentJob,
  upsertExportHubRecentJob,
  type ExportHubRecentJob,
} from '@/domains/exports/export_hub_recent';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';
import { resolveReportDisplayTitle } from '@/lib/report_paths';
import { confirmDestructiveAction } from '@/lib/mutation_audit';
import {
  type AdminValidationError,
  requireDateRange,
  requireInteger,
  requireNonEmpty,
  toastValidationError,
  validationError,
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

type ReportFormat = 'csv' | 'json';
type BillingFormat = 'csv' | 'ndjson';

function parseKind(value: string | null): ExportHubKind | undefined {
  if (value === 'report' || value === 'billing' || value === 'audit') {
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
  const { session } = useSession();
  const { data: catalog } = useResource((signal) => fetchReportCatalogCached(signal), []);

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
  const [draftReportFormat, setDraftReportFormat] = useState<ReportFormat | ''>(() =>
    urlFormat === 'csv' || urlFormat === 'json' ? urlFormat : ''
  );
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
  const [downloading, setDownloading] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [pollToken, setPollToken] = useState(0);
  const [auditExportTruncated, setAuditExportTruncated] = useState(false);
  const [asyncStatusPhase, setAsyncStatusPhase] = useState<ExportHubStatusPhase>('idle');
  const [formValidationError, setFormValidationError] = useState<AdminValidationError | undefined>();

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
    if (format === 'csv' || format === 'json') {
      setDraftReportFormat(format);
    }
    if (format === 'csv' || format === 'ndjson') {
      setDraftBillingFormat(format);
    }
    const rowLimit = searchParams.get('row_limit');
    if (rowLimit) {
      setDraftRowLimit(rowLimit);
    }
  }, [searchParams]);

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
  const autoPolling = Boolean(jobId) && selectedKind !== 'audit' && jobPending;
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
    };
    setRecentJobs(patchExportHubRecentJob(jobId, patch));

    const toastKey = `${jobId}:${normalizeExportJobStatus(status)}`;
    if (completionToastRef.current === toastKey) {
      return;
    }
    if (isJobReadyStatus(status)) {
      completionToastRef.current = toastKey;
      toast.success('Export ready for download');
      return;
    }
    if (exportJobPhase(status) === 'failed') {
      completionToastRef.current = toastKey;
      toast.error(jobErrorMessage ?? 'Export failed');
    }
  }, [job, jobErrorMessage, jobId]);

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
    (nextJobId: string, rowLimitValue: number, status = 'pending') => {
      const label = recentJobLabel(selectedKind, selectedEntry, draftReportKey.trim());
      setRecentJobs(
        upsertExportHubRecentJob({
          jobId: nextJobId,
          kind: selectedKind,
          label,
          customerId: draftCustomerId.trim() || undefined,
          rowLimit: rowLimitValue,
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

  const onDraftReportFormatChange = useCallback(
    (value: ReportFormat) => {
      clearFormValidationError();
      setDraftReportFormat(value);
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

  const onRunExport = useCallback(async () => {
    setAsyncStatusPhase('pending');
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
      const created = await createReportJob({
        customer_id: customerId,
        report_key: reportKey,
        from: fromIso,
        to: toIso,
        format: draftReportFormat as ReportFormat,
        row_limit: effectiveRowLimit,
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
      if (selectedEntry?.id && !selectedEntry.id.startsWith('custom-')) {
        next.set('entry', selectedEntry.id);
      }
      replaceSearchParams(next);
      setDraftJobId(nextId);
      recordRecentJob(nextId, effectiveRowLimit);
      setPollToken((value) => value + 1);
      toast.success('Report export job enqueued');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
      setAsyncStatusPhase('idle');
    } finally {
      setCreating(false);
    }
  }, [
    draftBillingFormat,
    draftCustomerId,
    draftFrom,
    draftRedactPii,
    draftReportFormat,
    draftReportKey,
    draftTo,
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
          triggerBlobDownload(blob, `${draftReportKey || 'report'}.csv`);
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
        triggerBlobDownload(blob, `${draftReportKey || 'report'}.${draftReportFormat}`);
      }
      setAsyncStatusPhase('idle');
      toast.success('Export downloaded');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
      setAsyncStatusPhase('idle');
    } finally {
      setDownloading(false);
    }
  }, [draftBillingFormat, draftJobId, draftReportFormat, draftReportKey, selectedKind]);

  return {
    catalogEntries,
    catalogPickerValue,
    selectedKind,
    draftCustomerId,
    draftReportKey,
    draftFrom,
    draftTo,
    draftReportFormat,
    draftBillingFormat,
    draftRedactPii,
    draftRowLimit,
    rowLimitBounds,
    rowLimit,
    draftJobId,
    job,
    jobPending,
    autoPolling,
    jobStartedAtMs,
    activeRowLimit,
    recentJobs,
    creating,
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
    onDraftReportFormatChange,
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
    onRetryRecentJob,
  };
}
