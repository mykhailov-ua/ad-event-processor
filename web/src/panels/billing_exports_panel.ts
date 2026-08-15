import type { ViewHandle } from '../lib/router_types.js';
import type { BillingExportJobDTO } from '../types/api/billing.js';
import { el, eventTargetValue, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { isoDaysAgo, toIsoNow } from '../helpers/date_presets.js';
import {
  createBillingExport,
  downloadBillingExport,
  pollBillingExportJob,
} from '../helpers/billing_admin_api.js';
import { renderSelect } from '../ui/select.js';
import { renderButton } from '../ui/button.js';

export type BillingExportsPanelOpts = {
  customerId: string;
  tenant: boolean;
};

type ExportJobRow = BillingExportJobDTO & { localId: string };

/**
 * Normalize export job status for comparisons.
 */
function exportJobStatus(job: Pick<BillingExportJobDTO, 'status'>): string {
  return String(job.status ?? '').trim().toUpperCase();
}

/**
 * Mount billing ledger export form and job list.
 */
export function mountBillingExportsPanel(
  container: HTMLElement,
  opts: BillingExportsPanelOpts,
): ViewHandle {
  let destroyed = false;
  let fromDate = isoDaysAgo(90).slice(0, 10);
  let toDate = toIsoNow().slice(0, 10);
  let format: 'csv' | 'ndjson' = 'csv';
  let submitting = false;
  let pollAbort: AbortController | null = null;
  const jobs: ExportJobRow[] = [];

  function render(): void {
    if (destroyed) return;
    const cid = opts.customerId;
    replaceChildren(container,
      !cid
        ? el('p', { className: 'text-muted text-sm' },
          opts.tenant
            ? 'Customer context missing.'
            : 'Enter customer_id above to export ledger lines.',
        )
        : el('div', { className: 'stack', 'data-testid': 'billing-exports-panel' },
          el('p', { className: 'text-muted text-sm' },
            'Exports ledger lines for the selected customer and date range (UTC). Large windows may take up to two minutes.',
          ),
          el('div', { className: 'filter-row' },
            el('label', { className: 'form-field' },
              'From',
              el('input', {
                type: 'date',
                className: 'form-input form-input--sm',
                value: fromDate,
                onChange: (e: Event) => { fromDate = eventTargetValue(e); },
              }),
            ),
            el('label', { className: 'form-field' },
              'To',
              el('input', {
                type: 'date',
                className: 'form-input form-input--sm',
                value: toDate,
                onChange: (e: Event) => { toDate = eventTargetValue(e); },
              }),
            ),
            el('label', { className: 'form-field' },
              'Format',
              renderSelect({
                value: format,
                options: [
                  { value: 'csv', label: 'CSV' },
                  { value: 'ndjson', label: 'NDJSON' },
                ],
                className: 'min-w-32',
                onChange: (v: string) => { format = v === 'ndjson' ? 'ndjson' : 'csv'; render(); },
              }),
            ),
            renderButton({
              label: submitting ? 'Creating…' : 'Create export',
              variant: 'primary',
              size: 'sm',
              loading: submitting,
              disabled: submitting || !cid,
              testId: 'billing-export-create',
              onClick: startExport,
            }),
          ),
          jobs.length > 0
            ? el('div', { className: 'table-wrapper elevation-raised' },
              el('table', { className: 'data-table' },
                el('thead', null,
                  el('tr', null,
                    el('th', { scope: 'col' }, 'Job'),
                    el('th', { scope: 'col' }, 'Status'),
                    el('th', { scope: 'col' }, 'Bytes'),
                    el('th', { scope: 'col' }, 'Created'),
                    el('th', { scope: 'col' }, ''),
                  ),
                ),
                el('tbody', null,
                  jobs.map((job) => el('tr', null,
                    el('td', { className: 'font-mono text-xs' }, job.id.slice(0, 8)),
                    el('td', null, exportJobStatus(job)),
                    el('td', { className: 'font-mono' }, String(job.bytes ?? '—')),
                    el('td', { className: 'text-muted text-xs' },
                      job.created_at ? new Date(job.created_at).toLocaleString() : '—',
                    ),
                    el('td', null,
                      exportJobStatus(job) === 'COMPLETED'
                        ? renderButton({
                          label: 'Download',
                          variant: 'secondary',
                          size: 'sm',
                          testId: `billing-export-download-${job.id}`,
                          onClick: () => downloadJob(job),
                        })
                        : exportJobStatus(job) === 'FAILED'
                          ? el('span', { className: 'text-danger text-xs' }, job.error ?? 'Failed')
                          : el('span', { className: 'text-muted text-xs' }, 'Running…'),
                    ),
                  )),
                ),
              ),
            )
            : null,
        ),
    );
  }

  async function downloadJob(job: ExportJobRow): Promise<void> {
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
  }

  async function pollJob(jobId: string, localId: string): Promise<void> {
    const signal = pollAbort?.signal;
    const [finalJob, pollErr] = await to(pollBillingExportJob(jobId, { signal }));
    if (destroyed) return;
    submitting = false;
    const idx = jobs.findIndex((j) => j.localId === localId);
    if (pollErr) {
      if (pollErr instanceof ConfirmCancelledError) return;
      const view = mapServiceError(pollErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      if (idx >= 0) {
        jobs[idx] = { ...jobs[idx], status: 'FAILED', error: view.message };
      }
      render();
      return;
    }
    if (idx >= 0 && finalJob) {
      jobs[idx] = { ...jobs[idx], ...finalJob };
    }
    if (finalJob?.status && exportJobStatus(finalJob) === 'COMPLETED') {
      pushToastMessage({ title: 'Export ready', message: 'Download is available below.' });
    }
    render();
  }

  async function startExport(): Promise<void> {
    const cid = opts.customerId;
    if (!cid || submitting) return;
    submitting = true;
    pollAbort?.abort();
    pollAbort = new AbortController();
    render();
    const [jobId, createErr] = await to(createBillingExport({
      customer_id: cid,
      from: fromDate,
      to: toDate,
      format,
    }));
    if (createErr) {
      submitting = false;
      if (createErr instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const view = mapServiceError(createErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    const localId = `local-${Date.now()}`;
    jobs.unshift({
      localId,
      id: jobId ?? '',
      customer_id: cid,
      format,
      status: 'PENDING',
      created_at: new Date().toISOString(),
    });
    pushToastMessage({ title: 'Export queued', message: jobId ?? '' });
    render();
    if (jobId) pollJob(jobId, localId);
  }

  render();

  return {
    destroy() {
      destroyed = true;
      pollAbort?.abort();
      pollAbort = null;
    },
  };
}
