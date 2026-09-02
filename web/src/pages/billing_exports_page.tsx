import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import {
  createBillingExportJob,
  downloadBillingExportJob,
  getBillingExportJob,
} from '@/api/billing_api';
import { BillingExports } from '@/domains/billing/billing_exports';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

function defaultRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to);
  from.setUTCDate(from.getUTCDate() - 7);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function BillingExportsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();
  const range = useMemo(() => defaultRange(), []);

  const jobId = searchParams.get('job_id') ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '',
  );
  const [draftFrom, setDraftFrom] = useState(
    toDatetimeLocalValue(searchParams.get('from') ?? range.from),
  );
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(searchParams.get('to') ?? range.to));
  const [draftFormat, setDraftFormat] = useState<'csv' | 'ndjson'>('csv');
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
      return getBillingExportJob(jobId, signal);
    },
    [jobId, pollToken],
  );

  const onCreateJob = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      return;
    }
    setCreating(true);
    setActionError(undefined);
    try {
      const created = await createBillingExportJob({
        customer_id: customerId,
        from: fromDatetimeLocalValue(draftFrom) ?? range.from,
        to: fromDatetimeLocalValue(draftTo) ?? range.to,
        format: draftFormat,
      });
      const nextId = created.job_id;
      if (!nextId) {
        throw new Error('job_id missing in create response');
      }
      const next = new URLSearchParams(searchParams);
      next.set('job_id', nextId);
      next.set('customer_id', customerId);
      setSearchParams(next, { replace: true });
      setPollToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftCustomerId, draftFormat, draftFrom, draftTo, range.from, range.to, searchParams, setSearchParams]);

  const onPollJob = useCallback(() => {
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

  const onDownloadJob = useCallback(async () => {
    if (!draftJobId.trim()) {
      return;
    }
    setActionError(undefined);
    try {
      const blob = await downloadBillingExportJob(draftJobId.trim());
      const ext = draftFormat === 'ndjson' ? 'ndjson' : 'csv';
      triggerBlobDownload(blob, `billing-export-${draftJobId.trim()}.${ext}`);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    }
  }, [draftFormat, draftJobId]);

  return (
    <BillingExports
      draftCustomerId={draftCustomerId}
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
      onDraftFromChange={setDraftFrom}
      onDraftToChange={setDraftTo}
      onDraftFormatChange={setDraftFormat}
      onDraftJobIdChange={setDraftJobId}
      onCreateJob={onCreateJob}
      onPollJob={onPollJob}
      onDownloadJob={onDownloadJob}
    />
  );
}
