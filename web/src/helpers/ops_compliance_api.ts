import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type {
  ConsentProofListResponse,
  ConsentRecordBody,
  RolesReloadResponse,
  SupportFeedbackCreateBody,
  SupportFeedbackCreateResponse,
  SupportFeedbackMetaDTO,
} from '../types/api/ops_compliance.js';

/** HMAC-SHA256 hex digest for consent webhook body (matches server verifier). */
export async function signConsentBody(secret: string, body: string): Promise<string> {
  const enc = new TextEncoder();
  const key = await crypto.subtle.importKey(
    'raw',
    enc.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  );
  const sig = await crypto.subtle.sign('HMAC', key, enc.encode(body));
  return Array.from(new Uint8Array(sig))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export async function fetchConsentProofs(
  userId = '',
  cursor = '',
  limit = 50,
): Promise<ConsentProofListResponse> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (userId) params.set('user_id', userId);
  if (cursor) params.set('cursor', cursor);
  const res = await api<ConsentProofListResponse>(`/api/v1/ops/consent/proofs?${params.toString()}`);
  return {
    items: res.data?.items ?? [],
    next_cursor: res.data?.next_cursor,
  };
}

export async function postConsentRecord(
  body: ConsentRecordBody,
  signature: string,
): Promise<void> {
  const json = JSON.stringify(body);
  await api('/api/v1/consent', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Consent-Signature': signature,
    },
    body: json,
  });
}

export async function fetchSupportFeedbackMeta(): Promise<SupportFeedbackMetaDTO> {
  const res = await api<SupportFeedbackMetaDTO>('/api/v1/support/feedback/meta');
  return res.data ?? { deployment_id: '', binary_version: '' };
}

export async function submitSupportFeedback(
  body: SupportFeedbackCreateBody,
): Promise<SupportFeedbackCreateResponse> {
  const res = await apiConfirmed<SupportFeedbackCreateResponse>('/api/v1/support/feedback', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data ?? { id: '' };
}

export async function reloadRoles(): Promise<RolesReloadResponse> {
  const res = await apiConfirmed<RolesReloadResponse>('/api/v1/ops/roles/reload', {
    method: 'POST',
  });
  return res.data ?? { status: '', path: '' };
}

export async function checkTlsAllowed(hostname: string, askToken?: string): Promise<boolean> {
  const path = `/api/v1/ops/domains/${encodeURIComponent(hostname.trim())}/tls-allowed`;
  const headers: Record<string, string> = {};
  const token = askToken?.trim();
  if (token) headers['X-Caddy-Ask-Token'] = token;
  try {
    const res = await api<{ allowed: boolean }>(path, { headers });
    return res.data?.allowed === true;
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) return false;
    throw e;
  }
}
