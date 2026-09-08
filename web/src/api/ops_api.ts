import { apiFetch, apiJson, apiJsonArray, parseApiError } from './client.js';
import type {
  DashboardMetrics,
  DashboardMetricsQuery,
  DashboardSummary,
  DlqInboxListQuery,
  DlqListQuery,
  DLQInboxListResponse,
  DLQListResponse,
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
  OpsConsentProofsResponse,
  OpsDomainRotationResponse,
  OpsMlModelEvalResponse,
  OpsMlModelStatusResponse,
  OpsRumResponse,
  OpsTlsAllowedResponse,
  OutboxListResponse,
  ReconListQuery,
  ReconRun,
  StackHealthSnapshot,
  StatusOKResponse,
} from './types.js';

export async function getStackHealthSnapshot(signal?: AbortSignal): Promise<StackHealthSnapshot> {
  return apiJson<StackHealthSnapshot>('/api/v1/ops/health/snapshot', { signal });
}

export async function getOpsDoctor(signal?: AbortSignal): Promise<DoctorSummary> {
  return apiJson<DoctorSummary>('/api/v1/ops/doctor', { signal });
}

export async function getOpsDashboardSummary(signal?: AbortSignal): Promise<DashboardSummary> {
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
  signal?: AbortSignal
): Promise<DLQInboxListResponse> {
  return apiJson<DLQInboxListResponse>(buildDlqInboxPath(params), { signal });
}

export function buildOpsDlqPath(params: DlqListQuery = {}): string {
  const search = new URLSearchParams();

  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }

  const query = search.toString();
  return query ? `/api/v1/ops/dlq?${query}` : '/api/v1/ops/dlq';
}

export async function listOpsDlq(
  params: DlqListQuery = {},
  signal?: AbortSignal
): Promise<DLQListResponse> {
  return apiJson<DLQListResponse>(buildOpsDlqPath(params), { signal });
}

export async function retryOpsDlqEntry(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/ops/dlq/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    signal,
  });

  if (!response.ok) {
    throw await parseApiError(response);
  }
}

export async function retryDlqInboxEntry(
  id: string,
  source: string,
  signal?: AbortSignal
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
    throw await parseApiError(response);
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
  signal?: AbortSignal
): Promise<OpsBlacklistListResponse> {
  return apiJson<OpsBlacklistListResponse>(buildBlacklistPath(params), { signal });
}

export async function addOpsBlacklistEntry(
  body: OpsBlacklistWriteRequest,
  signal?: AbortSignal
): Promise<void> {
  await apiJson<void>('/api/v1/ops/blacklist', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function removeOpsBlacklistEntry(
  body: OpsBlacklistDeleteRequest,
  signal?: AbortSignal
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
  signal?: AbortSignal
): Promise<OutboxListResponse> {
  return apiJson<OutboxListResponse>(buildOpsOutboxPath(params), { signal });
}

export async function listOpsShards(signal?: AbortSignal): Promise<OpsShardsResponse> {
  return apiJson<OpsShardsResponse>('/api/v1/ops/shards', { signal });
}

export async function triggerOpsShard0Catchup(
  signal?: AbortSignal
): Promise<OpsShardCatchupResponse> {
  return apiJson<OpsShardCatchupResponse>('/api/v1/ops/shards/0/catchup', {
    method: 'POST',
    signal,
  });
}

export async function getOpsMlModelStatus(signal?: AbortSignal): Promise<OpsMlModelStatusResponse> {
  return apiJson<OpsMlModelStatusResponse>('/api/v1/ops/ml-model', { signal });
}

export async function getOpsMlModelEval(signal?: AbortSignal): Promise<OpsMlModelEvalResponse> {
  return apiJson<OpsMlModelEvalResponse>('/api/v1/ops/ml-model/eval', { signal });
}

export async function listOpsMlLabels(signal?: AbortSignal): Promise<MLManualLabel[]> {
  return apiJsonArray<MLManualLabel>('/api/v1/ops/ml-model/labels', { signal });
}

export async function addOpsMlLabel(
  body: FraudManualLabelRequest,
  signal?: AbortSignal
): Promise<void> {
  await apiJson<void>('/api/v1/ops/ml-model/labels', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getOpsDomainRotation(
  signal?: AbortSignal
): Promise<OpsDomainRotationResponse> {
  return apiJson<OpsDomainRotationResponse>('/api/v1/ops/domains/rotation', { signal });
}

export async function checkOpsTlsAllowed(
  hostname: string,
  signal?: AbortSignal
): Promise<OpsTlsAllowedResponse> {
  const query = new URLSearchParams({ domain: hostname });
  return apiJson<OpsTlsAllowedResponse>(`/api/v1/ops/domains/tls-allowed?${query.toString()}`, {
    signal,
  });
}

export async function getOpsConsentProofs(signal?: AbortSignal): Promise<OpsConsentProofsResponse> {
  return apiJson<OpsConsentProofsResponse>('/api/v1/ops/consent/proofs', { signal });
}

export async function getOpsRum(signal?: AbortSignal): Promise<OpsRumResponse> {
  return apiJson<OpsRumResponse>('/api/v1/ops/rum', { signal });
}

export async function getOpsDashboardMetrics(
  params: DashboardMetricsQuery = {},
  signal?: AbortSignal
): Promise<DashboardMetrics> {
  const search = new URLSearchParams();
  if (params.range) {
    search.set('range', params.range);
  }
  const query = search.toString();
  const path = query ? `/api/v1/ops/dashboard/metrics?${query}` : '/api/v1/ops/dashboard/metrics';
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
  signal?: AbortSignal
): Promise<ReconRun[]> {
  return apiJsonArray<ReconRun>(buildReconRunsPath(params), { signal });
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

export async function postOpsSupportBundle(signal?: AbortSignal): Promise<OpsSupportBundleResult> {
  const response = await apiFetch('/api/v1/ops/support/bundle', {
    method: 'POST',
    signal,
  });

  if (!response.ok) {
    throw await parseApiError(response);
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
  onError?: (error: Error) => void
): () => void {
  const source = new EventSource('/api/v1/ops/dashboard/stream', { withCredentials: true });

  const handleDashboard = (event: MessageEvent<string>) => {
    try {
      const payload = JSON.parse(event.data) as { data?: DashboardSummary };
      if (payload.data) {
        onSummary(payload.data);
      }
    } catch (err: unknown) {
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
