import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type DomainHealthRow = {
  hostname: string;
  role: 'tracking' | 'admin' | 'custom';
  health_status: 'healthy' | 'degraded' | 'down' | 'unknown';
  ssl_status: 'valid' | 'expiring' | 'expired' | 'missing' | 'unknown';
  ssl_not_after?: string;
  http_status?: number;
  probe_latency_ms?: number;
  probe_detail?: string;
  last_probe_at?: string;
  updated_at?: string;
};

export type DomainSSLSetupResult = {
  hostname: string;
  status: string;
  message: string;
  output?: string;
};

export async function fetchDomains(): Promise<DomainHealthRow[]> {
  const res = await api<DomainHealthRow[]>('/api/v1/domains');
  return Array.isArray(res.data) ? res.data : [];
}

export async function addCustomDomain(hostname: string): Promise<DomainHealthRow> {
  const res = await apiConfirmed<DomainHealthRow>('/api/v1/domains', {
    method: 'POST',
    body: JSON.stringify({ hostname }),
  });
  return res.data;
}

export async function deleteCustomDomain(hostname: string): Promise<void> {
  await apiConfirmed(`/api/v1/domains/${encodeURIComponent(hostname)}`, {
    method: 'DELETE',
  });
}

export async function probeDomain(hostname: string): Promise<DomainHealthRow> {
  const res = await apiConfirmed<DomainHealthRow>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/probe`,
    { method: 'POST' },
  );
  return res.data;
}

export async function setupDomainSSL(hostname: string): Promise<DomainSSLSetupResult> {
  const res = await apiConfirmed<DomainSSLSetupResult>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/ssl/setup`,
    { method: 'POST' },
  );
  return res.data;
}

export function healthStatusLabel(status: string): string {
  switch (status) {
    case 'healthy': return 'Healthy';
    case 'degraded': return 'Degraded';
    case 'down': return 'Down';
    default: return 'Unknown';
  }
}

export function sslStatusLabel(status: string): string {
  switch (status) {
    case 'valid': return 'Valid';
    case 'expiring': return 'Expiring';
    case 'expired': return 'Expired';
    case 'missing': return 'Missing';
    default: return 'Unknown';
  }
}
