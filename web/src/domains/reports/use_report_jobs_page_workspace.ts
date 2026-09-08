// async report jobs: job_id in URL drives poll + download; create form drafts local until submit.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  cancelReportJob,
  createReportJob,
  downloadReportJob,
  getReportJob,
} from '@/api/reports_api';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { defaultReportRange } from '@/lib/report_paths';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';

export function useReportJobsPageWorkspace() {
  const [searchParams, { replaceSearchParams }] = useTransitionSearchParams();
  const { session } = useSession();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const { data: catalog } = useResource((signal) => fetchReportCatalogCached(signal), []);

  const jobId = searchParams.get('job_id') ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(
    searchParams.get('customer_id') ?? session?.default_customer_id ?? ''
  );
  const [draftReportKey, setDraftReportKey] = useState(
    searchParams.get('report_key') ?? 'placements'
  );
  const [draftFrom, setDraftFrom] = useState(
    toDatetimeLocalValue(searchParams.get('from') ?? defaultRange.from)
  );
  const [draftTo, setDraftTo] = useState(
    toDatetimeLocalValue(searchParams.get('to') ?? defaultRange.to)
  );
  const initialFormat = searchParams.get('format');
  const [draftFormat, setDraftFormat] = useState<'csv' | 'json'>(
    initialFormat === 'json' ? 'json' : 'csv'
  );
  const [draftJobId, setDraftJobId] = useState(jobId);
  const [creating, setCreating] = useState(false);
  const [polling, setPolling] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [pollToken, setPollToken] = useState(0);

  const reportKeyOptions = useMemo(() => {
    const keys = (catalog?.rows ?? [])
      .map((row) => row.key)
      .filter((key): key is string => Boolean(key?.trim()));
    if (keys.length === 0) {
      return ['placements', 'fraud-evidence-pack-bulk'];
    }
    return keys;
  }, [catalog?.rows]);

  useEffect(() => {
    setDraftJobId(jobId);
  }, [jobId]);

  useEffect(() => {
    const reportKey = searchParams.get('report_key');
    if (reportKey) {
      setDraftReportKey(reportKey);
    }
    const customerId = searchParams.get('customer_id');
    if (customerId) {
      setDraftCustomerId(customerId);
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
      setDraftFormat(format);
    }
  }, [searchParams]);

  const { data: job, error } = useResource(
    (signal) => {
      if (!jobId) {
        return Promise.resolve(undefined);
      }
      return getReportJob(jobId, signal);
    },
    [jobId, pollToken]
  );

  const onCreateJob = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    const reportKey = draftReportKey.trim();
    if (!customerId || !reportKey) {
      setActionError(new Error('Customer ID and report key are required'));
      return;
    }
    setCreating(true);
    setActionError(undefined);
    try {
      const created = await createReportJob({
        customer_id: customerId,
        report_key: reportKey,
        from: fromDatetimeLocalValue(draftFrom) ?? defaultRange.from,
        to: fromDatetimeLocalValue(draftTo) ?? defaultRange.to,
        format: draftFormat,
      });
      const nextId = created.id ?? created.job_id;
      if (!nextId) {
        throw new Error('job id missing in create response');
      }
      const next = new URLSearchParams(searchParams);
      next.set('job_id', nextId);
      next.set('customer_id', customerId);
      next.set('report_key', reportKey);
      replaceSearchParams(next);
      setPollToken((value) => value + 1);
      toast.success('Export job enqueued');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setCreating(false);
    }
  }, [
    defaultRange.from,
    defaultRange.to,
    draftCustomerId,
    draftFormat,
    draftFrom,
    draftReportKey,
    draftTo,
    replaceSearchParams,
    searchParams,
  ]);

  const onPollJob = useCallback(async () => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftJobId.trim();
    if (trimmed) {
      next.set('job_id', trimmed);
    } else {
      next.delete('job_id');
    }
    replaceSearchParams(next);
    setPolling(true);
    setPollToken((value) => value + 1);
    setPolling(false);
  }, [draftJobId, replaceSearchParams, searchParams]);

  const onCancelJob = useCallback(async () => {
    const trimmed = draftJobId.trim();
    if (!trimmed) {
      return;
    }
    if (!confirmDestructiveAction(`Cancel export job ${trimmed}?`)) {
      return;
    }
    setCancelling(true);
    setActionError(undefined);
    try {
      await cancelReportJob(trimmed);
      setPollToken((value) => value + 1);
      toast.success('Export job cancelled');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setCancelling(false);
    }
  }, [draftJobId]);

  const onDownloadJob = useCallback(async () => {
    const trimmed = draftJobId.trim();
    if (!trimmed) {
      return;
    }
    setDownloading(true);
    setActionError(undefined);
    try {
      const blob = await downloadReportJob(trimmed);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `${draftReportKey || 'report'}.${draftFormat}`;
      anchor.click();
      URL.revokeObjectURL(url);
      toast.success('Export downloaded');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setDownloading(false);
    }
  }, [draftFormat, draftJobId, draftReportKey]);

  return {
    draftCustomerId,
    draftReportKey,
    draftFrom,
    draftTo,
    draftFormat,
    draftJobId,
    job,
    reportKeyOptions,
    creating,
    polling,
    downloading,
    cancelling,
    error,
    actionError,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftReportKeyChange: setDraftReportKey,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftFormatChange: setDraftFormat,
    onDraftJobIdChange: setDraftJobId,
    onCreateJob,
    onPollJob,
    onCancelJob,
    onDownloadJob,
  };
}
