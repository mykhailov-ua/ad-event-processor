import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { components } from '../types/generated/openapi.js';

export type SmartAlertMetric = 'clicks' | 'cr' | 'roi_pct' | 'bot_clicks';
export type SmartAlertOperator = 'gt' | 'lt' | 'gte' | 'lte';

export type SmartAlertRule = components['schemas']['SmartAlertRule'];
export type SmartAlertEvent = components['schemas']['SmartAlertEvent'];
export type UpsertSmartAlertRuleBody = components['schemas']['UpsertSmartAlertRuleRequest'];

export const SMART_ALERT_METRICS: { value: SmartAlertMetric; label: string }[] = [
  { value: 'clicks', label: 'Clicks' },
  { value: 'cr', label: 'Conversion rate (%)' },
  { value: 'roi_pct', label: 'ROI (%)' },
  { value: 'bot_clicks', label: 'Bot / IVT clicks' },
];

export const SMART_ALERT_OPERATORS: { value: SmartAlertOperator; label: string }[] = [
  { value: 'gt', label: 'Greater than' },
  { value: 'gte', label: 'Greater or equal' },
  { value: 'lt', label: 'Less than' },
  { value: 'lte', label: 'Less or equal' },
];

/**
 * List smart alert rules for a customer.
 */
export async function fetchSmartAlertRules(customerId: string): Promise<SmartAlertRule[]> {
  const res = await api<SmartAlertRule[]>(
    `/api/v1/smart-alerts/rules?customer_id=${encodeURIComponent(customerId)}`
  );
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Create a smart alert rule.
 */
export async function createSmartAlertRule(
  body: UpsertSmartAlertRuleBody
): Promise<SmartAlertRule> {
  const res = await apiConfirmed<SmartAlertRule>('/api/v1/smart-alerts/rules', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data;
}

/**
 * Patch an existing smart alert rule.
 */
export async function updateSmartAlertRule(
  id: string,
  body: Omit<UpsertSmartAlertRuleBody, 'customer_id'> & { customer_id?: string }
): Promise<SmartAlertRule> {
  const res = await apiConfirmed<SmartAlertRule>(
    `/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    }
  );
  return res.data;
}

/**
 * Delete a smart alert rule.
 */
export async function deleteSmartAlertRule(id: string): Promise<void> {
  await apiConfirmed(`/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

/**
 * List recent smart alert firing history.
 */
export async function fetchSmartAlertHistory(
  customerId: string,
  limit = 50
): Promise<SmartAlertEvent[]> {
  const res = await api<SmartAlertEvent[]>(
    `/api/v1/smart-alerts/history?customer_id=${encodeURIComponent(customerId)}&limit=${limit}`
  );
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Acknowledge a fired smart alert event.
 */
export async function ackSmartAlertEvent(eventId: string): Promise<void> {
  await apiConfirmed(`/api/v1/smart-alerts/events/${encodeURIComponent(eventId)}/ack`, {
    method: 'POST',
  });
}
