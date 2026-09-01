import { apiFetch, apiJson, ApiError } from './client.js';
import type {
  DashboardMetrics,
  DashboardMetricsQuery,
  DashboardSummary,
  DlqInboxListQuery,
  DLQInboxListResponse,
  DoctorSummary,
  FraudManualLabelRequest,
  IncidentSnapshot,
  MLManualLabel,
  OpsBlacklistDeleteRequest,
  OpsBlacklistListQuery,
  OpsBlacklistListResponse,
  OpsBlacklistWriteRequest,
  OpsHomeSnapshot,
  OpsOutboxListQuery,
  OpsShardCatchupResponse,
  OpsShardsResponse,
  OutboxListResponse,
  ReconListQuery,
  ReconRun,
  StackHealthSnapshot,
  StatusOKResponse,
} from './types.js';

export async function getStackHealthSnapshot(
  signal?: AbortSignal,
): Promise<StackHealthSnapshot> {
  return apiJson<StackHealthSnapshot>('/api/v1/ops/health/snapshot', { signal });
}

export async function getOpsDoctor(signal?: AbortSignal): Promise<DoctorSummary> {
  return apiJson<DoctorSummary>('/api/v1/ops/doctor', { signal });
}

export async function getOpsDashboardSummary(
  signal?: AbortSignal,
): Promise<DashboardSummary> {
  return apiJson<DashboardSummary>('/api/v1/ops/dashboard/summary', { signal });
}

export async function fetchOpsHomeSnapshot(signal?: AbortSignal): Promise<OpsHomeSnapshot> {
  return apiJson<OpsHomeSnapshot>('/api/v1/ops/home', { signal });
}

export function buildDlqInboxPath(params: DlqInboxListQuery = {}): string {
  const search = new URLSearchParams();

  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }
  if (params.source) {
    search.set('source', params.source);
  }

  const query = search.toString();
  return query ? `/api/v1/ops/dlq/inbox?${query}` : '/api/v1/ops/dlq/inbox';
}

export async function listDlqInbox(
  params: DlqInboxListQuery = {},
  signal?: AbortSignal,
): Promise<DLQInboxListResponse> {
  return apiJson<DLQInboxListResponse>(buildDlqInboxPath(params), { signal });
}

export async function retryDlqInboxEntry(
  id: string,
  source: string,
  signal?: AbortSignal,
): Promise<void> {
  const idempotencyKey =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random()}`;

  const response = await apiFetch(`/api/v1/ops/dlq/inbox/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    headers: {
      'Idempotency-Key': idempotencyKey,
    },
    body: JSON.stringify({ source }),
    signal,
  });

  if (!response.ok) {
    let code = 'HTTP_ERROR';
    let message = response.statusText || `HTTP ${response.status}`;
    try {
      const body: unknown = await response.json();
      if (body && typeof body === 'object') {
        const record = body as Record<string, unknown>;
        const errorField = record.error;
        if (errorField && typeof errorField === 'object') {
          const errObj = errorField as Record<string, unknown>;
          if (typeof errObj.code === 'string') {
            code = errObj.code;
          }
          if (typeof errObj.message === 'string') {
            message = errObj.message;
          }
        }
      }
    } catch {
      // Non-JSON error body.
    }
    throw new ApiError(response.status, code, message);
  }
}

export function buildBlacklistPath(params: OpsBlacklistListQuery = {}): string {
  const search = new URLSearchParams();
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  return query ? `/api/v1/ops/blacklist?${query}` : '/api/v1/ops/blacklist';
}

export async function listOpsBlacklist(
  params: OpsBlacklistListQuery = {},
  signal?: AbortSignal,
): Promise<OpsBlacklistListResponse> {
  return apiJson<OpsBlacklistListResponse>(buildBlacklistPath(params), { signal });
}

export async function addOpsBlacklistEntry(
  body: OpsBlacklistWriteRequest,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<void>('/api/v1/ops/blacklist', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function removeOpsBlacklistEntry(
  body: OpsBlacklistDeleteRequest,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<void>('/api/v1/ops/blacklist', {
    method: 'DELETE',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getOpsIncidents(signal?: AbortSignal): Promise<IncidentSnapshot> {
  return apiJson<IncidentSnapshot>('/api/v1/ops/incidents', { signal });
}

export function buildOpsOutboxPath(params: OpsOutboxListQuery = {}): string {
  const search = new URLSearchParams();
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }
  const query = search.toString();
  return query ? `/api/v1/ops/outbox?${query}` : '/api/v1/ops/outbox';
}

export async function listOpsOutbox(
  params: OpsOutboxListQuery = {},
  signal?: AbortSignal,
): Promise<OutboxListResponse> {
  return apiJson<OutboxListResponse>(buildOpsOutboxPath(params), { signal });
}

export async function listOpsShards(signal?: AbortSignal): Promise<OpsShardsResponse> {
  return apiJson<OpsShardsResponse>('/api/v1/ops/shards', { signal });
}

export async function triggerOpsShard0Catchup(
  signal?: AbortSignal,
): Promise<OpsShardCatchupResponse> {
  return apiJson<OpsShardCatchupResponse>('/api/v1/ops/shards/0/catchup', {
    method: 'POST',
    signal,
  });
}

export async function getOpsMlModelStatus(
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/ops/ml-model', { signal });
}

export async function getOpsMlModelEval(
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/ops/ml-model/eval', { signal });
}

export async function listOpsMlLabels(signal?: AbortSignal): Promise<MLManualLabel[]> {
  return apiJson<MLManualLabel[]>('/api/v1/ops/ml-model/labels', { signal });
}

export async function addOpsMlLabel(
  body: FraudManualLabelRequest,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<void>('/api/v1/ops/ml-model/labels', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getOpsDomainRotation(
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/ops/domains/rotation', { signal });
}

export async function getOpsTlsAllowedList(
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/ops/domains/tls-allowed', { signal });
}

export async function getOpsTlsAllowedHost(
  hostname: string,
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>(
    `/api/v1/ops/domains/${encodeURIComponent(hostname)}/tls-allowed`,
    { signal },
  );
}

export async function getOpsConsentProofs(
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/ops/consent/proofs', { signal });
}

export async function getOpsRum(signal?: AbortSignal): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>('/api/v1/ops/rum', { signal });
}

export async function getOpsDashboardMetrics(
  params: DashboardMetricsQuery = {},
  signal?: AbortSignal,
): Promise<DashboardMetrics> {
  const search = new URLSearchParams();
  if (params.range) {
    search.set('range', params.range);
  }
  const query = search.toString();
  const path = query
    ? `/api/v1/ops/dashboard/metrics?${query}`
    : '/api/v1/ops/dashboard/metrics';
  return apiJson<DashboardMetrics>(path, { signal });
}

export function buildReconRunsPath(params: ReconListQuery = {}): string {
  const search = new URLSearchParams();
  if (params.service) {
    search.set('service', params.service);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  return query ? `/api/v1/recon/runs?${query}` : '/api/v1/recon/runs';
}

export async function listReconRuns(
  params: ReconListQuery = {},
  signal?: AbortSignal,
): Promise<ReconRun[]> {
  return apiJson<ReconRun[]>(buildReconRunsPath(params), { signal });
}

export async function reloadOpsRoles(signal?: AbortSignal): Promise<StatusOKResponse> {
  return apiJson<StatusOKResponse>('/api/v1/ops/roles/reload', {
    method: 'POST',
    signal,
  });
}

export type OpsSupportBundleResult = {
  blob: Blob;
  filename: string;
};

const defaultSupportBundleFilename = 'ad-event-processor-support-bundle.tar.gz';

function parseContentDispositionFilename(header: string | null): string | undefined {
  if (!header) {
    return undefined;
  }
  const match = /filename="([^"]+)"/.exec(header);
  return match?.[1];
}

export async function postOpsSupportBundle(
  signal?: AbortSignal,
): Promise<OpsSupportBundleResult> {
  const response = await apiFetch('/api/v1/ops/support/bundle', {
    method: 'POST',
    signal,
  });

  if (!response.ok) {
    let code = 'HTTP_ERROR';
    let message = response.statusText || `HTTP ${response.status}`;
    try {
      const body: unknown = await response.json();
      if (body && typeof body === 'object') {
        const record = body as Record<string, unknown>;
        const errorField = record.error;
        if (errorField && typeof errorField === 'object') {
          const errObj = errorField as Record<string, unknown>;
          if (typeof errObj.code === 'string') {
            code = errObj.code;
          }
          if (typeof errObj.message === 'string') {
            message = errObj.message;
          }
        }
      }
    } catch {
      // Binary or empty body on error.
    }
    throw new ApiError(response.status, code, message);
  }

  return {
    blob: await response.blob(),
    filename:
      parseContentDispositionFilename(response.headers.get('Content-Disposition')) ??
      defaultSupportBundleFilename,
  };
}

export function subscribeOpsDashboardStream(
  onSummary: (summary: DashboardSummary) => void,
  onError?: (error: Error) => void,
): () => void {
  const source = new EventSource('/api/v1/ops/dashboard/stream', { withCredentials: true });

  const handleDashboard = (event: MessageEvent<string>) => {
    try {
      const payload = JSON.parse(event.data) as { data?: DashboardSummary };
      if (payload.data) {
        onSummary(payload.data);
      }
    } catch (err) {
      onError?.(err instanceof Error ? err : new Error(String(err)));
    }
  };

  source.addEventListener('dashboard', handleDashboard as EventListener);
  source.onerror = () => {
    onError?.(new Error('Dashboard stream disconnected'));
  };

  return () => {
    source.removeEventListener('dashboard', handleDashboard as EventListener);
    source.close();
  };
}
