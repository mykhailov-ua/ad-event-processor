import { apiFetch, apiJson, ApiError } from './client.js';
import type {
  AcceptEulaRequest,
  ApplyLicenseRequest,
  ConsentRecord,
  CreateSupportFeedbackRequest,
  DisputeListQuery,
  DisputeListResponse,
  EulaStatus,
  LicenseStatus,
  MetaResponse,
  SupportFeedbackMeta,
  SupportFeedbackResponse,
} from './types.js';

export async function getMeta(signal?: AbortSignal): Promise<MetaResponse> {
  return apiJson<MetaResponse>('/api/v1/meta', { signal });
}

export async function getEulaStatus(signal?: AbortSignal): Promise<EulaStatus> {
  return apiJson<EulaStatus>('/api/v1/eula', { signal });
}

export async function acceptEula(
  body: AcceptEulaRequest,
  signal?: AbortSignal,
): Promise<EulaStatus> {
  return apiJson<EulaStatus>('/api/v1/eula/accept', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getLicenseStatus(signal?: AbortSignal): Promise<LicenseStatus> {
  return apiJson<LicenseStatus>('/api/v1/license/status', { signal });
}

export async function applyLicense(
  body: ApplyLicenseRequest,
  signal?: AbortSignal,
): Promise<LicenseStatus> {
  return apiJson<LicenseStatus>('/api/v1/license/apply', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listDisputes(
  params: DisputeListQuery = {},
  signal?: AbortSignal,
): Promise<DisputeListResponse> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  const path = query ? `/api/v1/disputes?${query}` : '/api/v1/disputes';
  return apiJson<DisputeListResponse>(path, { signal });
}

export async function getSupportFeedbackMeta(signal?: AbortSignal): Promise<SupportFeedbackMeta> {
  return apiJson<SupportFeedbackMeta>('/api/v1/support/feedback/meta', { signal });
}

export async function createSupportFeedback(
  body: CreateSupportFeedbackRequest,
  signal?: AbortSignal,
): Promise<SupportFeedbackResponse> {
  return apiJson<SupportFeedbackResponse>('/api/v1/support/feedback', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function postConsent(
  body: ConsentRecord,
  signature: string,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch('/api/v1/consent', {
    method: 'POST',
    headers: {
      'X-Consent-Signature': signature,
    },
    body: JSON.stringify(body),
    signal,
  });

  if (!response.ok) {
    let code = 'HTTP_ERROR';
    let message = response.statusText || `HTTP ${response.status}`;
    try {
      const payload: unknown = await response.json();
      if (payload && typeof payload === 'object') {
        const record = payload as Record<string, unknown>;
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
