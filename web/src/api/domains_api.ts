import { apiFetch, apiJson } from './client.js';
import type {
  AddDomainRequest,
  DomainHealth,
  DomainSSLSetupResult,
  ParkDomainRequest,
  ParkDomainResponse,
} from './types.js';

export async function listDomains(signal?: AbortSignal): Promise<DomainHealth[]> {
  return apiJson<DomainHealth[]>('/api/v1/domains', { signal });
}

export async function addDomain(body: AddDomainRequest, signal?: AbortSignal): Promise<DomainHealth> {
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
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function probeDomain(hostname: string, signal?: AbortSignal): Promise<DomainHealth> {
  return apiJson<DomainHealth>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/probe`,
    { method: 'POST', signal },
  );
}

export async function setupDomainSsl(
  hostname: string,
  signal?: AbortSignal,
): Promise<DomainSSLSetupResult> {
  return apiJson<DomainSSLSetupResult>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/ssl/setup`,
    { method: 'POST', signal },
  );
}

export async function parkDomain(
  body: ParkDomainRequest,
  signal?: AbortSignal,
): Promise<ParkDomainResponse> {
  return apiJson<ParkDomainResponse>('/api/v1/domains/park', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
