import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { components } from '../types/generated/openapi.js';

export type DomainHealthRow = components['schemas']['DomainHealth'];
export type DomainSSLSetupResult = components['schemas']['DomainSSLSetupResult'];
export type ParkDomainRequest = components['schemas']['ParkDomainRequest'];
export type ParkDomainResponse = components['schemas']['ParkDomainResponse'];

/**
 * List domain health rows (tracking, admin, custom).
 */
export async function fetchDomains(): Promise<DomainHealthRow[]> {
  const res = await api<DomainHealthRow[]>('/api/v1/domains');
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Register a custom tracking hostname.
 */
export async function addCustomDomain(hostname: string): Promise<DomainHealthRow> {
  const res = await apiConfirmed<DomainHealthRow>('/api/v1/domains', {
    method: 'POST',
    body: JSON.stringify({ hostname }),
  });
  return res.data;
}

/**
 * Remove a custom domain from health tracking.
 */
export async function deleteCustomDomain(hostname: string): Promise<void> {
  await apiConfirmed(`/api/v1/domains/${encodeURIComponent(hostname)}`, {
    method: 'DELETE',
  });
}

/**
 * Run an immediate HTTP/TLS probe for one hostname.
 */
export async function probeDomain(hostname: string): Promise<DomainHealthRow> {
  const res = await apiConfirmed<DomainHealthRow>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/probe`,
    { method: 'POST' }
  );
  return res.data;
}

/**
 * Trigger appliance TLS setup for a hostname.
 */
export async function setupDomainSSL(hostname: string): Promise<DomainSSLSetupResult> {
  const res = await apiConfirmed<DomainSSLSetupResult>(
    `/api/v1/domains/${encodeURIComponent(hostname)}/ssl/setup`,
    { method: 'POST' }
  );
  return res.data;
}

/**
 * Park a domain via Cloudflare DNS integration.
 */
export async function parkDomain(req: ParkDomainRequest): Promise<ParkDomainResponse> {
  const res = await apiConfirmed<ParkDomainResponse>('/api/v1/domains/park', {
    method: 'POST',
    body: JSON.stringify(req),
  });
  return (
    res.data ?? {
      success: false,
      dns_record_id: '',
      ssl_status: '',
    }
  );
}

/** Map health_status wire value to operator label. */
export function healthStatusLabel(status: string): string {
  switch (status) {
    case 'healthy':
      return 'Healthy';
    case 'degraded':
      return 'Degraded';
    case 'down':
      return 'Down';
    default:
      return 'Unknown';
  }
}

/** Map ssl_status wire value to operator label. */
export function sslStatusLabel(status: string): string {
  switch (status) {
    case 'valid':
      return 'Valid';
    case 'expiring':
      return 'Expiring';
    case 'expired':
      return 'Expired';
    case 'missing':
      return 'Missing';
    default:
      return 'Unknown';
  }
}
