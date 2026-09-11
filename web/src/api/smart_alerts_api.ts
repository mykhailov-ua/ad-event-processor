import { apiFetch, apiJson, parseApiError } from './client.js';
import type {
  SmartAlertEvent,
  SmartAlertRule,
  SmartAlertsListHistoryQuery,
  SmartAlertsListRulesQuery,
  UpsertSmartAlertRuleRequest,
} from './types.js';

export async function listSmartAlertRules(
  params: SmartAlertsListRulesQuery,
  signal?: AbortSignal
): Promise<SmartAlertRule[]> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  const query = search.toString();
  const path = query ? `/api/v1/smart-alerts/rules?${query}` : '/api/v1/smart-alerts/rules';
  return apiJson<SmartAlertRule[]>(path, { signal });
}

export async function createSmartAlertRule(
  body: UpsertSmartAlertRuleRequest,
  signal?: AbortSignal
): Promise<SmartAlertRule> {
  return apiJson<SmartAlertRule>('/api/v1/smart-alerts/rules', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateSmartAlertRule(
  id: string,
  body: UpsertSmartAlertRuleRequest,
  signal?: AbortSignal
): Promise<SmartAlertRule> {
  return apiJson<SmartAlertRule>(`/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteSmartAlertRule(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw await parseApiError(response);
  }
}

export async function listSmartAlertHistory(
  params: SmartAlertsListHistoryQuery,
  signal?: AbortSignal
): Promise<SmartAlertEvent[]> {
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
  const path = query ? `/api/v1/smart-alerts/history?${query}` : '/api/v1/smart-alerts/history';
  return apiJson<SmartAlertEvent[]>(path, { signal });
}

export async function ackSmartAlertEvent(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/smart-alerts/events/${encodeURIComponent(id)}/ack`, {
    method: 'POST',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw await parseApiError(response);
  }
}
