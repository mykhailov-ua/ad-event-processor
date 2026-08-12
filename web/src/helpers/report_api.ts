import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { probeStart, probeEnd } from './perf_probe.js';
import { stubReportPath } from '../models/report.js';
import { to } from '../lib/to.js';

export type StubProbeResult = {
  ok: boolean;
  status: number;
  stub: boolean;
  message: string;
  path: string;
};

export type ReportJobPollResult = {
  ok: boolean;
  status: string;
  message: string;
};

export type ReportExportSubmitResult = {
  ok: boolean;
  status: number;
  stub: boolean;
  message: string;
  jobId?: string;
  rateLimited?: boolean;
};

export type SavedViewInput = {
  customerId: string;
  name: string;
  reportKey: string;
  spec: Record<string, unknown>;
};

export type SavedViewRow = {
  report_key?: string;
  customer_id?: string;
  spec?: Record<string, unknown> | string;
};

export type PollReportJobOpts = {
  intervalMs?: number;
  maxAttempts?: number;
  signal?: AbortSignal;
};

export type SubmitReportExportSpec = {
  customerId: string;
  reportKey: string;
  from: string;
  to: string;
  signal?: AbortSignal;
};

/**
 * Parse Retry-After header from an API error into milliseconds.
 */
export function parseRetryAfterMs(err: {
  status?: number;
  responseHeaders?: Headers | null;
} | null | undefined): number {
  const raw = err?.responseHeaders?.get?.('Retry-After');
  const sec = raw ? Number.parseInt(raw, 10) : 0;
  if (Number.isFinite(sec) && sec > 0) return sec * 1000;
  return 2500;
}

/**
 * Pause until the given number of milliseconds elapse.
 */
function sleepMs(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new Error('aborted'));
      return;
    }
    const timer = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener('abort', () => {
        clearTimeout(timer);
        reject(new Error('aborted'));
      }, { once: true });
    }
  });
}

/**
 * Probe a planned report endpoint (expected 501 until backend ships).
 */
export async function probeStubReport(
  reportKey: string,
  customerId = '',
): Promise<StubProbeResult> {
  const path = stubReportPath(reportKey);
  if (!path) {
    return { ok: false, status: 0, stub: false, message: 'unknown report key', path: '' };
  }
  const probe = probeStart('report.stub.fetch');
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  const qs = params.toString();
  const url = qs ? `${path}?${qs}` : path;
  const [, err] = await to(api(url));
  if (!err) {
    probeEnd(probe, { allocs: 1, bytes: 64 });
    return { ok: true, status: 200, stub: false, message: 'live', path: url };
  }
  probeEnd(probe, { allocs: 1, bytes: 128 });
  const apiErr = err instanceof ApiError ? err : null;
  const status = apiErr?.status ?? 0;
  const stub = status === 501 || apiErr?.code === 'NOT_IMPLEMENTED' || apiErr?.stub === true;
  return {
    ok: false,
    status,
    stub,
    message: err.message ?? String(err),
    path: url,
  };
}

/**
 * Download a completed report export job as CSV.
 */
export async function downloadReportExport(jobId: string, filename = 'report.csv'): Promise<void> {
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
 */
export async function pollReportJob(
  jobId: string,
  opts: PollReportJobOpts = {},
): Promise<ReportJobPollResult> {
  const intervalMs = opts.intervalMs ?? 1500;
  const maxAttempts = opts.maxAttempts ?? 20;
  let rateLimitHits = 0;
  for (let i = 0; i < maxAttempts; i++) {
    if (opts.signal?.aborted) {
      return { ok: false, status: 'ABORTED', message: 'polling aborted' };
    }
    const [res, err] = await to(api(`/api/v1/reports/jobs/${jobId}`, { signal: opts.signal }));
    if (err) {
      const apiErr = err instanceof ApiError ? err : null;
      if (apiErr?.status === 429 && rateLimitHits < 5) {
        rateLimitHits += 1;
        const wait = parseRetryAfterMs(apiErr);
        try {
          await sleepMs(wait, opts.signal);
        } catch {
          return { ok: false, status: 'ABORTED', message: 'polling aborted' };
        }
        i -= 1;
        continue;
      }
      return { ok: false, status: 'ERROR', message: err.message ?? String(err) };
    }
    const payload = res?.data as { status?: string; error?: string } | null | undefined;
    const status = payload?.status ?? '';
    if (status === 'COMPLETED' || status === 'FAILED') {
      return {
        ok: status === 'COMPLETED',
        status,
        message: status === 'COMPLETED' ? 'Export ready' : (payload?.error ?? 'Export failed'),
      };
    }
    try {
      await sleepMs(intervalMs, opts.signal);
    } catch {
      return { ok: false, status: 'ABORTED', message: 'polling aborted' };
    }
  }
  return { ok: false, status: 'TIMEOUT', message: 'Export job polling timed out' };
}

/**
 * Submit an async report export job.
 */
export async function submitReportExport(
  spec: SubmitReportExportSpec,
): Promise<ReportExportSubmitResult> {
  const probe = probeStart('report.export.submit');
  const body = {
    customer_id: spec.customerId,
    report_key: spec.reportKey,
    from: spec.from,
    to: spec.to,
    format: 'csv',
  };
  const [res, err] = await to(api('/api/v1/reports/jobs', {
    method: 'POST',
    body: JSON.stringify(body),
    idempotencyScope: `report-export:${spec.customerId}:${spec.reportKey}`,
    signal: spec.signal,
  }));
  if (!err) {
    probeEnd(probe, { allocs: 1, bytes: 96 });
    const payload = res?.data as { job_id?: string; id?: string } | null | undefined;
    return {
      ok: true,
      status: 201,
      stub: false,
      message: 'export job created',
      jobId: payload?.job_id ?? payload?.id,
    };
  }
  const apiErr = err instanceof ApiError ? err : null;
  if (apiErr?.status === 429) {
    probeEnd(probe, { allocs: 1, bytes: 128 });
    return {
      ok: false,
      status: 429,
      stub: false,
      rateLimited: true,
      message: `Rate limited — retry in ${Math.ceil(parseRetryAfterMs(apiErr) / 1000)}s`,
    };
  }
  probeEnd(probe, { allocs: 1, bytes: 128 });
  const status = apiErr?.status ?? 0;
  const stub = status === 501 || apiErr?.code === 'NOT_IMPLEMENTED';
  return {
    ok: false,
    status,
    stub,
    message: err.message ?? String(err),
  };
}

/**
 * List saved report views for a customer.
 */
export async function listSavedViews(customerId: string): Promise<unknown[]> {
  const probe = probeStart('views.list');
  const params = new URLSearchParams({ customer_id: customerId });
  const { data } = await api(`/api/v1/views?${params.toString()}`);
  const payload = data as unknown[] | { items?: unknown[] } | null | undefined;
  const items = Array.isArray(payload) ? payload : (payload?.items ?? []);
  probeEnd(probe, { allocs: 1, bytes: items.length * 120 });
  return items;
}

/**
 * Create a saved report view preset.
 */
export async function createSavedView(input: SavedViewInput): Promise<unknown> {
  const probe = probeStart('views.create');
  const { data } = await apiConfirmed('/api/v1/views', {
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

/**
 * Delete a saved report view preset.
 */
export async function deleteSavedView(viewId: string): Promise<void> {
  await apiConfirmed(`/api/v1/views/${encodeURIComponent(viewId)}`, { method: 'DELETE' });
}

/**
 * Build report route href from a saved view row.
 */
export function savedViewHref(view: SavedViewRow): string {
  const base = `/reports/${view.report_key ?? 'placements'}`;
  let spec: Record<string, unknown> = {};
  if (view.spec) {
    try {
      spec = typeof view.spec === 'string'
        ? (JSON.parse(view.spec) as Record<string, unknown>)
        : view.spec;
    } catch {
      spec = {};
    }
  }
  const qs = new URLSearchParams();
  if (view.customer_id) qs.set('customer_id', view.customer_id);
  if (spec.from) qs.set('from', String(spec.from));
  if (spec.to) qs.set('to', String(spec.to));
  const q = qs.toString();
  return q ? `${base}?${q}` : base;
}
