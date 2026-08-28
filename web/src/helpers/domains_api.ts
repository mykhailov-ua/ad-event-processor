import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type DomainHealth = {
  hostname?: string;
  role?: string;
  health_status?: string;
  ssl_status?: string;
  ssl_not_after?: string;
  http_status?: number;
  probe_latency_ms?: number;
  probe_detail?: string;
  last_probe_at?: string;
  updated_at?: string;
};

export type DomainSslSetupResult = {
  hostname?: string;
  status?: string;
  message?: string;
  output?: string;
};

export type ParkDomainRequest = {
  domain: string;
  cloudflare_zone_id: string;
  pool_id?: string;
};

export type ParkDomainResponse = {
  success?: boolean;
  dns_record_id?: string;
  ssl_status?: string;
  hostname?: string;
  pool_id?: string;
};

export async function fetchDomains(signal?: AbortSignal): Promise<DomainHealth[]> {
  const result = await api<DomainHealth[]>('/api/v1/domains', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function addDomain(hostname: string): Promise<DomainHealth> {
  const result = await apiConfirmed<DomainHealth>('/api/v1/domains', {
    method: 'POST',
    body: JSON.stringify({ hostname }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('add domain failed');
  }
  return result.data ?? {};
}

export async function deleteDomain(hostname: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/domains/${encodeURIComponent(hostname)}`, {
    method: 'DELETE',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete domain failed');
  }
}

export async function probeDomain(hostname: string): Promise<DomainHealth> {
  const result = await apiConfirmed<DomainHealth>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/probe`,
    { method: 'POST', body: '{}' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('domain probe failed');
  }
  return result.data ?? {};
}

export async function setupDomainSsl(hostname: string): Promise<DomainSslSetupResult> {
  const result = await apiConfirmed<DomainSslSetupResult>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/ssl/setup`,
    { method: 'POST', body: '{}' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('ssl setup failed');
  }
  return result.data ?? {};
}

export async function parkDomain(body: ParkDomainRequest): Promise<ParkDomainResponse> {
  const result = await apiConfirmed<ParkDomainResponse>('/api/v1/domains/park', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('park domain failed');
  }
  return result.data ?? {};
}
