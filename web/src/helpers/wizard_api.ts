import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type WizardStep = 'traffic_source' | 'integration_template' | 'flow_skeleton' | 'budget';

export type WizardSession = {
  session_id?: string;
  step?: string;
  customer_id?: string;
  payload?: Record<string, unknown>;
};

export type WizardSessionRequest = {
  action: 'create' | 'update' | 'commit';
  session_id?: string;
  customer_id?: string;
  template_key?: string;
  step?: WizardStep;
  payload?: Record<string, unknown>;
  idempotency_key?: string;
  publish?: boolean;
};

export async function postWizardSession(body: WizardSessionRequest): Promise<unknown> {
  const result = await apiConfirmed('/api/v1/campaigns/wizard/session', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('wizard session failed');
  }
  return result.data;
}

export async function getWizardSession(
  sessionId: string,
  signal?: AbortSignal
): Promise<WizardSession> {
  const qs = new URLSearchParams({ session_id: sessionId });
  const result = await api<WizardSession>(`/api/v1/campaigns/wizard/session?${qs.toString()}`, {
    signal,
  });
  return (result.data ?? {}) as WizardSession;
}
