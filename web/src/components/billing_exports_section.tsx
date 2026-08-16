import { useEffect, useRef, useState } from 'react';
import type { BillingExportJobDTO } from '../types/api/billing.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { isoDaysAgo, toIsoNow } from '../helpers/date_presets.js';
import {
  createBillingExport,
  downloadBillingExport,
  pollBillingExportJob,
} from '../helpers/billing_admin_api.js';
import { to } from '../lib/to.js';
import { Button } from './button.js';

export type BillingExportsSectionProps = {
  customerId: string;
  tenant: boolean;
};

type ExportJobRow = BillingExportJobDTO & { localId: string };

function exportJobStatus(job: Pick<BillingExportJobDTO, 'status'>): string {
  return String(job.status ?? '').trim().toUpperCase();
}

/**
 * Billing ledger export form and job list.
 */
export function BillingExportsSection({ customerId, tenant }: BillingExportsSectionProps) {
  const [fromDate, setFromDate] = useState(() => isoDaysAgo(90).slice(0, 10));
  const [toDate, setToDate] = useState(() => toIsoNow().slice(0, 10));
  const [format, setFormat] = useState<'csv' | 'ndjson'>('csv');
  const [submitting, setSubmitting] = useState(false);
  const [jobs, setJobs] = useState<ExportJobRow[]>([]);
  const pollAbortRef = useRef<AbortController | null>(null);

  useEffect(() => () => {
    pollAbortRef.current?.abort();
    pollAbortRef.current = null;
  }, []);

  const downloadJob = async (job: ExportJobRow) => {
    const ext = job.format === 'ndjson' ? 'ndjson' : 'csv';
    const [, err] = await to(downloadBillingExport(
      job.id,
      `ledger-${job.customer_id}.${ext}`,
      job.download_url ?? '',
    ));
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    }
  };

  const pollJob = async (jobId: string, localId: string) => {
    const signal = pollAbortRef.current?.signal;
    const [finalJob, pollErr] = await to(pollBillingExportJob(jobId, { signal }));
    setSubmitting(false);
    if (pollErr) {
      if (pollErr instanceof ConfirmCancelledError) return;
      const view = mapServiceError(pollErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      setJobs((prev) => prev.map((j) => (
        j.localId === localId ? { ...j, status: 'FAILED', error: view.message } : j
      )));
      return;
    }
    setJobs((prev) => prev.map((j) => (
      j.localId === localId && finalJob ? { ...j, ...finalJob } : j
    )));
    if (finalJob?.status && exportJobStatus(finalJob) === 'COMPLETED') {
      pushToastMessage({ title: 'Export ready', message: 'Download is available below.' });
    }
  };

  const startExport = async () => {
    const cid = customerId;
    if (!cid || submitting) return;
    setSubmitting(true);
    pollAbortRef.current?.abort();
    pollAbortRef.current = new AbortController();
    const [jobId, createErr] = await to(createBillingExport({
      customer_id: cid,
      from: fromDate,
      to: toDate,
      format,
    }));
    if (createErr) {
      setSubmitting(false);
      if (createErr instanceof ConfirmCancelledError) return;
      const view = mapServiceError(createErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    const localId = `local-${Date.now()}`;
    setJobs((prev) => [{
      localId,
      id: jobId ?? '',
      customer_id: cid,
      format,
      status: 'PENDING',
      created_at: new Date().toISOString(),
    }, ...prev]);
    pushToastMessage({ title: 'Export queued', message: jobId ?? '' });
    if (jobId) void pollJob(jobId, localId);
  };

  if (!customerId) {
    return (
      <p className="text-muted text-sm">
        {tenant
          ? 'Customer context missing.'
          : 'Enter customer_id above to export ledger lines.'}
      </p>
    );
  }

  return (
    <div className="stack" data-testid="billing-exports-panel">
      <p className="text-muted text-sm">
        Exports ledger lines for the selected customer and date range (UTC).
        Large windows may take up to two minutes.
      </p>
      <div className="filter-row">
        <label className="form-field">
          From
          <input
            type="date"
            className="form-input form-input--sm"
            value={fromDate}
            onChange={(e) => setFromDate(e.target.value)}
          />
        </label>
        <label className="form-field">
          To
          <input
            type="date"
            className="form-input form-input--sm"
            value={toDate}
            onChange={(e) => setToDate(e.target.value)}
          />
        </label>
        <label className="form-field">
          Format
          <select
            className="form-input form-input--sm min-w-32"
            value={format}
            onChange={(e) => setFormat(e.target.value === 'ndjson' ? 'ndjson' : 'csv')}
          >
            <option value="csv">CSV</option>
            <option value="ndjson">NDJSON</option>
          </select>
        </label>
        <Button
          label={submitting ? 'Creating…' : 'Create export'}
          variant="primary"
          size="sm"
          loading={submitting}
          disabled={submitting}
          data-testid="billing-export-create"
          onClick={() => void startExport()}
        />
      </div>
      {jobs.length > 0 ? (
        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Job</th>
                <th scope="col">Status</th>
                <th scope="col">Bytes</th>
                <th scope="col">Created</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => {
                const status = exportJobStatus(job);
                return (
                  <tr key={job.localId}>
                    <td className="font-mono text-xs">{job.id.slice(0, 8)}</td>
                    <td>{status}</td>
                    <td className="font-mono">{String(job.bytes ?? '—')}</td>
                    <td className="text-muted text-xs">
                      {job.created_at ? new Date(job.created_at).toLocaleString() : '—'}
                    </td>
                    <td>
                      {status === 'COMPLETED' ? (
                        <Button
                          label="Download"
                          variant="secondary"
                          size="sm"
                          data-testid={`billing-export-download-${job.id}`}
                          onClick={() => void downloadJob(job)}
                        />
                      ) : status === 'FAILED' ? (
                        <span className="text-danger text-xs">{job.error ?? 'Failed'}</span>
                      ) : (
                        <span className="text-muted text-xs">Running…</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  );
}
