import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import {
  cancelReportJob,
  createReportJob,
  downloadReportJob,
  getReportJob,
} from '@/api/reports_api';
import { ReportJobs } from '@/domains/reports/report_jobs';
import { useResource } from '@/hooks/use_resource';
import { useSession } from '@/hooks/use_session';
import { defaultReportRange } from '@/lib/report_paths';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

export function ReportJobsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const jobId = searchParams.get('job_id') ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '',
  );
  const [draftReportKey, setDraftReportKey] = useState(
    searchParams.get('report_key') ?? 'placements',
  );
  const [draftFrom, setDraftFrom] = useState(
    toDatetimeLocalValue(searchParams.get('from') ?? defaultRange.from),
  );
  const [draftTo, setDraftTo] = useState(
    toDatetimeLocalValue(searchParams.get('to') ?? defaultRange.to),
  );
  const [draftFormat, setDraftFormat] = useState<'csv' | 'json'>('csv');
  const [draftJobId, setDraftJobId] = useState(jobId);
  const [creating, setCreating] = useState(false);
  const [polling, setPolling] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [pollToken, setPollToken] = useState(0);

  useEffect(() => {
    setDraftJobId(jobId);
  }, [jobId]);

  const { data: job, error } = useResource(
    (signal) => {
      if (!jobId) {
        return Promise.resolve(undefined);
      }
      return getReportJob(jobId, signal);
    },
    [jobId, pollToken],
  );

  const onCreateJob = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    const reportKey = draftReportKey.trim();
    if (!customerId || !reportKey) {
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
      setSearchParams(next, { replace: true });
      setPollToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
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
    searchParams,
    setSearchParams,
  ]);

  const onPollJob = useCallback(async () => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftJobId.trim();
    if (trimmed) {
      next.set('job_id', trimmed);
    } else {
      next.delete('job_id');
    }
    setSearchParams(next, { replace: true });
    setPolling(true);
    setPollToken((value) => value + 1);
    setPolling(false);
  }, [draftJobId, searchParams, setSearchParams]);

  const onCancelJob = useCallback(async () => {
    if (!draftJobId.trim()) {
      return;
    }
    setActionError(undefined);
    try {
      await cancelReportJob(draftJobId.trim());
      setPollToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    }
  }, [draftJobId]);

  const onDownloadJob = useCallback(async () => {
    if (!draftJobId.trim()) {
      return;
    }
    setActionError(undefined);
    try {
      const blob = await downloadReportJob(draftJobId.trim());
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `${draftReportKey || 'report'}.${draftFormat}`;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    }
  }, [draftFormat, draftJobId, draftReportKey]);

  return (
    <ReportJobs
      draftCustomerId={draftCustomerId}
      draftReportKey={draftReportKey}
      draftFrom={draftFrom}
      draftTo={draftTo}
      draftFormat={draftFormat}
      draftJobId={draftJobId}
      job={job}
      creating={creating}
      polling={polling}
      error={error}
      actionError={actionError}
      onDraftCustomerIdChange={setDraftCustomerId}
      onDraftReportKeyChange={setDraftReportKey}
      onDraftFromChange={setDraftFrom}
      onDraftToChange={setDraftTo}
      onDraftFormatChange={setDraftFormat}
      onDraftJobIdChange={setDraftJobId}
      onCreateJob={onCreateJob}
      onPollJob={onPollJob}
      onCancelJob={onCancelJob}
      onDownloadJob={onDownloadJob}
    />
  );
}
