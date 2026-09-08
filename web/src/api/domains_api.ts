import { apiFetch, apiJson, apiJsonArray, parseApiError } from './client.js';
import type {
  AddDomainRequest,
  BurnDomainRequest,
  BurnDomainResponse,
  CloudflareZone,
  DomainBulkJobStatus,
  DomainBulkRequest,
  DomainHealth,
  DomainSSLSetupResult,
  ParkDomainRequest,
  ParkDomainResponse,
  WildcardSSLRequest,
  WildcardSSLResponse,
} from './types.js';

export type DomainHealthFilter = 'all' | 'healthy' | 'degraded' | 'burned';

export type DomainListQuery = {
  health_filter?: DomainHealthFilter;
};

export async function listDomains(
  query?: DomainListQuery,
  signal?: AbortSignal
): Promise<DomainHealth[]> {
  const params = new URLSearchParams();
  if (query?.health_filter && query.health_filter !== 'all') {
    params.set('health_filter', query.health_filter);
  }
  const qs = params.toString();
  const path = qs ? `/api/v1/domains?${qs}` : '/api/v1/domains';
  return apiJsonArray<DomainHealth>(path, { signal });
}

export async function addDomain(
  body: AddDomainRequest,
  signal?: AbortSignal
): Promise<DomainHealth> {
  return apiJson<DomainHealth>('/api/v1/domains', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteDomain(hostname: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/domains/${encodeURIComponent(hostname)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
}

export async function probeDomain(hostname: string, signal?: AbortSignal): Promise<DomainHealth> {
  return apiJson<DomainHealth>(`/api/v1/domains/${encodeURIComponent(hostname)}/probe`, {
    method: 'POST',
    signal,
  });
}

export async function setupDomainSsl(
  hostname: string,
  signal?: AbortSignal
): Promise<DomainSSLSetupResult> {
  return apiJson<DomainSSLSetupResult>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/ssl/setup`,
    { method: 'POST', signal }
  );
}

export async function parkDomain(
  body: ParkDomainRequest,
  signal?: AbortSignal
): Promise<ParkDomainResponse> {
  return apiJson<ParkDomainResponse>('/api/v1/domains/park', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listCloudflareZones(signal?: AbortSignal): Promise<CloudflareZone[]> {
  return apiJsonArray<CloudflareZone>('/api/v1/domains/cloudflare/zones', { signal });
}

export async function setupWildcardSSL(
  body: WildcardSSLRequest,
  signal?: AbortSignal
): Promise<WildcardSSLResponse> {
  return apiJson<WildcardSSLResponse>('/api/v1/ops/domains/wildcard-ssl', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function startBulkDomainPark(
  body: DomainBulkRequest,
  signal?: AbortSignal
): Promise<DomainBulkJobStatus> {
  return apiJson<DomainBulkJobStatus>('/api/v1/ops/domains/bulk', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function startBulkDomainSSL(
  body: DomainBulkRequest,
  signal?: AbortSignal
): Promise<DomainBulkJobStatus> {
  return apiJson<DomainBulkJobStatus>('/api/v1/ops/domains/bulk-ssl', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getDomainBulkJob(
  jobId: string,
  signal?: AbortSignal
): Promise<DomainBulkJobStatus> {
  return apiJson<DomainBulkJobStatus>(`/api/v1/ops/domains/jobs/${encodeURIComponent(jobId)}`, {
    signal,
  });
}

export async function burnDomain(
  hostname: string,
  body: BurnDomainRequest,
  signal?: AbortSignal
): Promise<BurnDomainResponse> {
  return apiJson<BurnDomainResponse>(`/api/v1/ops/domains/${encodeURIComponent(hostname)}/burn`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}
