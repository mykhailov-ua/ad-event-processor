import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type DashboardServiceCard = {
  id?: string;
  name?: string;
  status?: string;
  detail?: string;
};

export type DashboardSummary = {
  generated_at?: string;
  services?: DashboardServiceCard[];
  drift_micro_max?: number;
  drift_alert?: boolean;
  rps_estimate?: number;
  outbox_pending?: number;
  emergency_breaker?: string;
};

export type DashboardMetricPoint = {
  name?: string;
  labels_hash?: string;
  ts?: string;
  value?: number;
};

export type DashboardMetrics = {
  range?: string;
  bucket_sec?: number;
  points?: DashboardMetricPoint[];
  generated_at?: string;
};

export type DoctorCheck = {
  id?: string;
  status?: string;
  message?: string;
  hint?: string;
  latency_ms?: number;
};

export type OpsDoctor = {
  overall?: string;
  checks?: DoctorCheck[];
  click_url_template?: string;
  tracking_domain?: string;
  rtb_mode?: string;
  rtb_enabled?: boolean;
};

export type OutboxEvent = {
  id?: number;
  event_type?: string;
  status?: string;
  created_at?: string;
};

export type OutboxList = {
  items?: OutboxEvent[];
  total?: number;
  next_cursor?: string;
};

export type FanOutSourceError = {
  source?: string;
  code?: string;
};

export type DLQEntry = {
  id: string;
  shard_id: number;
  stream_id: string;
  entry_id: string;
  campaign_id?: string;
  event_type?: string;
  error?: string;
  failed_at: string;
  retry_count: number;
  worker_id?: string;
};

export type DLQList = {
  items?: DLQEntry[];
  partial?: boolean;
  errors?: FanOutSourceError[];
  next_cursor?: string;
};

export type DLQInboxEntry = {
  id?: string;
  source?: string;
  campaign_id?: string;
  event_type?: string;
  error?: string;
  failed_at?: string;
  status?: string;
  retry_count?: number;
  shard_id?: number;
  stream_id?: string;
  entry_id?: string;
  click_id?: string;
  provider?: string;
};

export type DLQInbox = {
  items?: DLQInboxEntry[];
  next_cursor?: string;
  partial?: boolean;
  errors?: FanOutSourceError[];
};

export type ShardHealthStatus = {
  shard_id?: number;
  ping_ok?: boolean;
  ping_error?: string;
  ping_latency_ms?: number;
  config_version?: number | null;
  config_version_lag?: number;
  config_version_synced?: boolean;
};

export type ShardStatus = {
  shards?: ShardHealthStatus[];
  partial?: boolean;
  errors?: FanOutSourceError[];
};

export type DomainRotationHost = {
  hostname?: string;
  role?: string;
  health_status?: string;
  ssl_status?: string;
  pool_id?: string;
  pool_domain_status?: string;
  dmr_campaign_count?: number;
  active_campaign_count?: number;
};

export type DomainRotation = {
  hosts?: DomainRotationHost[];
};

export type DomainTlsAllowed = {
  allowed?: boolean;
};

export type BlacklistEntry = {
  id?: number;
  ip?: string;
  reason?: string;
  created_at?: string;
  expires_at?: string;
};

export type BlacklistList = {
  items?: BlacklistEntry[];
  total?: number;
};

export type ReconRun = {
  service?: string;
  id?: number;
  period_start?: string;
  period_end?: string;
  status?: string;
  total_delta?: number;
  campaigns_checked?: number;
  discrepancies_found?: number;
  findings_count?: number;
  intents_checked?: number;
  error_message?: string;
  created_at?: string;
  completed_at?: string;
};

export type ConsentProof = {
  id?: number;
  user_id_hash?: string;
  purposes?: number;
  source?: string;
  recorded_at?: string;
  ad_storage?: boolean;
  analytics_storage?: boolean;
};

export type ConsentProofList = {
  items?: ConsentProof[];
  next_cursor?: string;
};

export type MLModelVersion = {
  id?: string;
  artifact_hash?: string;
  status?: string;
  created_at?: string;
  artifact_metadata?: unknown;
};

export type MLModelRedis = {
  version_id?: string;
  hash?: string;
  applied_at?: string;
  shards_reporting?: number;
  shards_consistent?: boolean;
};

export type MLShardSync = {
  shard_id?: number;
  model_version?: string;
  phase?: string;
  started_at?: string;
};

export type MLFeatureImportance = {
  name?: string;
  value?: number;
};

export type MLModelStatus = {
  active_version?: MLModelVersion;
  syncing_version?: MLModelVersion;
  redis?: MLModelRedis;
  shard_sync?: MLShardSync[];
  drift?: unknown;
  drift_detected?: boolean;
  precision?: number;
  recall?: number;
  importance?: MLFeatureImportance[];
};

export type MLEvalMetricsBlock = {
  status?: string;
  label_method?: string;
  label_definition?: string;
  labeled_rows?: number;
  matched_rows?: number;
  confidence?: string;
  precision?: number;
  recall?: number;
  f1?: number;
  false_positive_rate?: number;
  tp?: number;
  fp?: number;
  fn?: number;
  tn?: number;
};

export type MLEvalReport = {
  status?: string;
  generated_at?: string;
  hours?: number;
  threshold?: number;
  proxy_metrics?: MLEvalMetricsBlock;
  audited_metrics?: MLEvalMetricsBlock;
  drift?: unknown;
  drift_detected?: boolean;
};

export type MLManualLabel = {
  ip_hash?: string;
  label?: number;
  reason?: string;
  source?: string;
  created_at?: string;
};

export type MLManualLabelBody = {
  ip_hash: string;
  label: number;
  reason?: string;
};

export type MutationPreview = {
  would_apply?: boolean;
  detail?: string;
};

function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

export function buildDashboardSummaryUrl(): string {
  return '/api/v1/ops/dashboard/summary';
}

export function buildDashboardMetricsUrl(rangeHours: number, metricName?: string): string {
  const qs = new URLSearchParams({ range: `${rangeHours}h` });
  if (metricName) qs.set('name', metricName);
  return `/api/v1/ops/dashboard/metrics?${qs.toString()}`;
}

export function buildDoctorUrl(): string {
  return '/api/v1/ops/doctor';
}

export function buildOutboxListUrl(params: {
  limit: number;
  cursor?: string;
  status?: string;
  event_type?: string;
}): string {
  const qs = new URLSearchParams({ limit: String(params.limit) });
  if (params.cursor) qs.set('cursor', params.cursor);
  if (params.status) qs.set('status', params.status);
  if (params.event_type) qs.set('event_type', params.event_type);
  return `/api/v1/ops/outbox?${qs.toString()}`;
}

export function buildDlqListUrl(params: { limit: number; cursor?: string }): string {
  const qs = new URLSearchParams({ limit: String(params.limit) });
  if (params.cursor) qs.set('cursor', params.cursor);
  return `/api/v1/ops/dlq?${qs.toString()}`;
}

export function buildDlqInboxUrl(params: {
  limit: number;
  cursor?: string;
  source?: string;
}): string {
  const qs = new URLSearchParams({ limit: String(params.limit) });
  if (params.cursor) qs.set('cursor', params.cursor);
  if (params.source) qs.set('source', params.source);
  return `/api/v1/ops/dlq/inbox?${qs.toString()}`;
}

export function buildShardsUrl(): string {
  return '/api/v1/ops/shards';
}

export function buildDomainRotationUrl(): string {
  return '/api/v1/ops/domains/rotation';
}

export function buildDomainTlsAllowedUrl(hostname?: string): string {
  if (hostname) {
    return `/api/v1/ops/domains/${encodeURIComponent(hostname)}/tls-allowed`;
  }
  return '/api/v1/ops/domains/tls-allowed';
}

export function buildBlacklistListUrl(params: { limit: number; offset: number }): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
  });
  return `/api/v1/ops/blacklist?${qs.toString()}`;
}

export function buildReconRunsUrl(params: {
  limit: number;
  offset: number;
  service?: string;
}): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
  });
  if (params.service) qs.set('service', params.service);
  return `/api/v1/recon/runs?${qs.toString()}`;
}

export function buildConsentProofsUrl(params: {
  limit: number;
  cursor?: string;
  user_id?: string;
}): string {
  const qs = new URLSearchParams({ limit: String(params.limit) });
  if (params.cursor) qs.set('cursor', params.cursor);
  if (params.user_id) qs.set('user_id', params.user_id);
  return `/api/v1/ops/consent/proofs?${qs.toString()}`;
}

export function buildMlModelUrl(): string {
  return '/api/v1/ops/ml-model';
}

export function buildMlEvalUrl(): string {
  return '/api/v1/ops/ml-model/eval';
}

export function buildMlLabelsUrl(): string {
  return '/api/v1/ops/ml-model/labels';
}

export async function fetchDashboardSummary(signal?: AbortSignal): Promise<DashboardSummary> {
  const result = await api<DashboardSummary>(buildDashboardSummaryUrl(), { signal });
  return result.data ?? {};
}

export async function fetchDashboardMetrics(
  rangeHours: number,
  metricName?: string,
  signal?: AbortSignal
): Promise<DashboardMetrics> {
  const result = await api<DashboardMetrics>(buildDashboardMetricsUrl(rangeHours, metricName), {
    signal,
  });
  return result.data ?? {};
}

export async function fetchOpsDoctor(signal?: AbortSignal): Promise<OpsDoctor> {
  const result = await api<OpsDoctor>(buildDoctorUrl(), { signal });
  return result.data ?? {};
}

export async function fetchOutboxList(
  params: { limit: number; cursor?: string; status?: string; event_type?: string },
  signal?: AbortSignal
): Promise<OutboxList> {
  const result = await api<OutboxList>(buildOutboxListUrl(params), { signal });
  return result.data ?? {};
}

export async function fetchDlqList(
  params: { limit: number; cursor?: string },
  signal?: AbortSignal
): Promise<DLQList> {
  const result = await api<DLQList>(buildDlqListUrl(params), { signal });
  return result.data ?? {};
}

export async function fetchDlqInbox(
  params: { limit: number; cursor?: string; source?: string },
  signal?: AbortSignal
): Promise<DLQInbox> {
  const result = await api<DLQInbox>(buildDlqInboxUrl(params), { signal });
  return result.data ?? {};
}

export async function fetchShardStatus(signal?: AbortSignal): Promise<ShardStatus> {
  const result = await api<ShardStatus>(buildShardsUrl(), { signal });
  return result.data ?? {};
}

export async function fetchDomainRotation(signal?: AbortSignal): Promise<DomainRotation> {
  const result = await api<DomainRotation>(buildDomainRotationUrl(), { signal });
  return result.data ?? {};
}

export async function fetchDomainTlsAllowed(
  hostname?: string,
  signal?: AbortSignal
): Promise<DomainTlsAllowed> {
  const result = await api<DomainTlsAllowed>(buildDomainTlsAllowedUrl(hostname), { signal });
  return result.data ?? {};
}

export async function fetchBlacklist(
  params: { limit: number; offset: number },
  signal?: AbortSignal
): Promise<BlacklistList> {
  const result = await api<BlacklistList>(buildBlacklistListUrl(params), { signal });
  const data = result.data ?? {};
  const totalHeader = result.headers.get('X-Total-Count');
  const parsed = totalHeader ? Number.parseInt(totalHeader, 10) : NaN;
  const total = Number.isFinite(parsed) ? parsed : data.total ?? data.items?.length ?? 0;
  return { items: data.items ?? [], total };
}

export async function fetchReconRuns(
  params: { limit: number; offset: number; service?: string },
  signal?: AbortSignal
): Promise<{ items: ReconRun[]; total: number }> {
  const result = await api<ReconRun[]>(buildReconRunsUrl(params), { signal });
  const items = Array.isArray(result.data) ? result.data : [];
  const totalHeader = result.headers.get('X-Total-Count');
  const parsed = totalHeader ? Number.parseInt(totalHeader, 10) : NaN;
  const total = Number.isFinite(parsed) ? parsed : items.length;
  return { items, total };
}

export async function fetchConsentProofs(
  params: { limit: number; cursor?: string; user_id?: string },
  signal?: AbortSignal
): Promise<ConsentProofList> {
  const result = await api<ConsentProofList>(buildConsentProofsUrl(params), { signal });
  return result.data ?? {};
}

export async function fetchMlModelStatus(signal?: AbortSignal): Promise<MLModelStatus> {
  const result = await api<MLModelStatus>(buildMlModelUrl(), { signal });
  return result.data ?? {};
}

export async function fetchMlEvalReport(signal?: AbortSignal): Promise<MLEvalReport> {
  const result = await api<MLEvalReport>(buildMlEvalUrl(), { signal });
  return result.data ?? {};
}

export async function fetchMlLabels(signal?: AbortSignal): Promise<MLManualLabel[]> {
  const result = await api<MLManualLabel[]>(buildMlLabelsUrl(), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export function isApiStubError(error: unknown): boolean {
  return error instanceof ApiError && (error.stub || error.status === 501);
}

export async function retryDlq(
  id: string,
  body?: { shard_id?: number; stream?: string; entry_id?: string }
): Promise<void> {
  const result = await apiConfirmed(`/api/v1/ops/dlq/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    headers: { 'Idempotency-Key': newIdempotencyKey() },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('DLQ retry failed');
  }
}

export async function retryDlqInbox(id: string, source: string): Promise<void> {
  const qs = new URLSearchParams({ source });
  const result = await apiConfirmed(
    `/api/v1/ops/dlq/inbox/${encodeURIComponent(id)}/retry?${qs.toString()}`,
    {
      method: 'POST',
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('DLQ inbox retry failed');
  }
}

export async function shardCatchup(): Promise<void> {
  const result = await apiConfirmed('/api/v1/ops/shards/0/catchup', { method: 'POST' });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('Shard catch-up failed');
  }
}

export async function reloadRoles(): Promise<{ status?: string; path?: string }> {
  const result = await apiConfirmed<{ status?: string; path?: string }>(
    '/api/v1/ops/roles/reload',
    { method: 'POST' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('Roles reload failed');
  }
  return result.data ?? {};
}

export async function blockBlacklist(
  body: { ip: string; source: string; ttl_seconds?: number },
  dryRun = false
): Promise<MutationPreview | void> {
  const headers: Record<string, string> = {};
  if (dryRun) headers['X-Dry-Run'] = '1';
  const result = await apiConfirmed<MutationPreview>('/api/v1/ops/blacklist', {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('Block IP failed');
  }
  if (dryRun) return result.data ?? {};
}

export async function unblockBlacklist(body: { ip: string; source: string }): Promise<void> {
  const result = await apiConfirmed('/api/v1/ops/blacklist', {
    method: 'DELETE',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('Unblock IP failed');
  }
}

export async function postMlLabels(body: MLManualLabelBody): Promise<void> {
  const result = await apiConfirmed('/api/v1/ops/ml-model/labels', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('Save ML label failed');
  }
}
