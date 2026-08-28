import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type SupportFeedbackMeta = {
  deployment_id?: string;
  binary_version?: string;
};

export type SupportFeedbackRequest = {
  type: string;
  contact_email?: string;
  message: string;
  attach_bundle?: boolean;
};

export type SupportFeedbackResponse = {
  id?: string;
};

export async function fetchSupportFeedbackMeta(
  signal?: AbortSignal
): Promise<SupportFeedbackMeta> {
  const result = await api<SupportFeedbackMeta>('/api/v1/support/feedback/meta', { signal });
  return result.data ?? {};
}

export async function submitSupportFeedback(
  body: SupportFeedbackRequest
): Promise<SupportFeedbackResponse> {
  const result = await apiConfirmed<SupportFeedbackResponse>('/api/v1/support/feedback', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('feedback submit failed');
  }
  return result.data ?? {};
}
