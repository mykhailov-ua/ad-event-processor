import { api } from './api_client.js';
import { probeStart, probeEnd } from './perf_probe.js';
import { stubReportPath } from '../models/report.js';

/**
 * Probe a planned report endpoint (expected 501 until backend ships).
 *
 * @param {string} reportKey
 * @param {string} [customerId]
 * @returns {Promise<{ ok: boolean, status: number, stub: boolean, message: string, path: string }>}
 */
export async function probeStubReport(reportKey, customerId = '') {
  const path = stubReportPath(reportKey);
  if (!path) {
    return { ok: false, status: 0, stub: false, message: 'unknown report key', path: '' };
  }
  const probe = probeStart('report.stub.fetch');
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  const qs = params.toString();
  const url = qs ? `${path}?${qs}` : path;
  try {
    await api(url);
    probeEnd(probe, { allocs: 1, bytes: 64 });
    return { ok: true, status: 200, stub: false, message: 'unexpected success', path: url };
  } catch (err) {
    probeEnd(probe, { allocs: 1, bytes: 128 });
    const status = err?.status ?? 0;
    const stub = status === 501 || err?.code === 'NOT_IMPLEMENTED' || err?.stub === true;
    return {
      ok: false,
      status,
      stub,
      message: err?.message ?? String(err),
      path: url,
    };
  }
}

/**
 * Download a completed report export job as CSV.
 *
 * @param {string} jobId
 * @param {string} [filename]
 * @returns {Promise<void>}
 */
export async function downloadReportExport(jobId, filename = 'report.csv') {
  const { apiBlob } = await import('./api_blob.js');
  const blob = await apiBlob(`/api/v1/reports/jobs/${jobId}/download`);
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

/**
 * Poll report export job status until terminal state.
 *
 * @param {string} jobId
 * @param {{ intervalMs?: number, maxAttempts?: number, signal?: AbortSignal }} [opts]
 * @returns {Promise<{ ok: boolean, status: string, message: string }>}
 */
export async function pollReportJob(jobId, opts = {}) {
  const intervalMs = opts.intervalMs ?? 1500;
  const maxAttempts = opts.maxAttempts ?? 20;
  for (let i = 0; i < maxAttempts; i++) {
    if (opts.signal?.aborted) {
      return { ok: false, status: 'ABORTED', message: 'polling aborted' };
    }
    try {
      const { data } = await api(`/api/v1/reports/jobs/${jobId}`, { signal: opts.signal });
      const status = data?.status ?? '';
      if (status === 'COMPLETED' || status === 'FAILED') {
        return {
          ok: status === 'COMPLETED',
          status,
          message: status === 'COMPLETED' ? 'Export ready' : (data?.error ?? 'Export failed'),
        };
      }
    } catch (err) {
      return { ok: false, status: 'ERROR', message: err?.message ?? String(err) };
    }
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  return { ok: false, status: 'TIMEOUT', message: 'Export job polling timed out' };
}

/**
 * Submit an async report export job (stub until backend ships).
 *
 * @param {{ customerId: string, reportKey: string, from: string, to: string }} spec
 * @returns {Promise<{ ok: boolean, status: number, stub: boolean, message: string, jobId?: string }>}
 */
export async function submitReportExport(spec) {
  const probe = probeStart('report.export.submit');
  const body = {
    customer_id: spec.customerId,
    report_key: spec.reportKey,
    from: spec.from,
    to: spec.to,
    format: 'csv',
  };
  try {
    const { data } = await api('/api/v1/reports/jobs', {
      method: 'POST',
      body: JSON.stringify(body),
      idempotencyScope: `report-export:${spec.customerId}:${spec.reportKey}`,
    });
    probeEnd(probe, { allocs: 1, bytes: 96 });
    return {
      ok: true,
      status: 201,
      stub: false,
      message: 'export job created',
      jobId: data?.job_id ?? data?.id,
    };
  } catch (err) {
    probeEnd(probe, { allocs: 1, bytes: 128 });
    const status = err?.status ?? 0;
    const stub = status === 501 || err?.code === 'NOT_IMPLEMENTED';
    return {
      ok: false,
      status,
      stub,
      message: err?.message ?? String(err),
    };
  }
}

/**
 * List saved report views for a customer.
 *
 * @param {string} customerId
 * @returns {Promise<object[]>}
 */
export async function listSavedViews(customerId) {
  const probe = probeStart('views.list');
  const params = new URLSearchParams({ customer_id: customerId });
  const { data } = await api(`/api/v1/views?${params.toString()}`);
  const items = Array.isArray(data) ? data : (data?.items ?? []);
  probeEnd(probe, { allocs: 1, bytes: items.length * 120 });
  return items;
}

/**
 * Create a saved report view preset.
 *
 * @param {{ customerId: string, name: string, reportKey: string, spec: object }} input
 * @returns {Promise<object>}
 */
export async function createSavedView(input) {
  const probe = probeStart('views.create');
  const { data } = await api('/api/v1/views', {
    method: 'POST',
    body: JSON.stringify({
      customer_id: input.customerId,
      name: input.name,
      report_key: input.reportKey,
      spec: input.spec,
      is_shared: false,
    }),
  });
  probeEnd(probe, { allocs: 1, bytes: 160 });
  return data;
}
