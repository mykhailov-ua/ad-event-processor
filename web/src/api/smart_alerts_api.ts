import { apiFetch, apiJson } from './client.js';
import type {
  SmartAlertEvent,
  SmartAlertsListHistoryQuery,
  SmartAlertsListRulesQuery,
  SmartAlertRule,
  UpsertSmartAlertRuleRequest,
} from './types.js';

export async function listSmartAlertRules(
  params: SmartAlertsListRulesQuery,
  signal?: AbortSignal,
): Promise<SmartAlertRule[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  return apiJson<SmartAlertRule[]>(`/api/v1/smart-alerts/rules?${search.toString()}`, { signal });
}

export async function listSmartAlertHistory(
  params: SmartAlertsListHistoryQuery,
  signal?: AbortSignal,
): Promise<SmartAlertEvent[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  return apiJson<SmartAlertEvent[]>(`/api/v1/smart-alerts/history?${search.toString()}`, {
    signal,
  });
}

export async function createSmartAlertRule(
  body: UpsertSmartAlertRuleRequest,
  signal?: AbortSignal,
): Promise<SmartAlertRule> {
  return apiJson<SmartAlertRule>('/api/v1/smart-alerts/rules', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateSmartAlertRule(
  ruleId: string,
  body: UpsertSmartAlertRuleRequest,
  signal?: AbortSignal,
): Promise<SmartAlertRule> {
  return apiJson<SmartAlertRule>(`/api/v1/smart-alerts/rules/${encodeURIComponent(ruleId)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteSmartAlertRule(ruleId: string, signal?: AbortSignal): Promise<void> {
  await apiJson<void>(`/api/v1/smart-alerts/rules/${encodeURIComponent(ruleId)}`, {
    method: 'DELETE',
    signal,
  });
}

export async function ackSmartAlertEvent(eventId: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(
    `/api/v1/smart-alerts/events/${encodeURIComponent(eventId)}/ack`,
    { method: 'POST', signal },
  );
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}
