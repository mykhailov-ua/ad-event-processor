import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type SmartAlertMetric = 'clicks' | 'cr' | 'roi_pct' | 'bot_clicks';
export type SmartAlertOperator = 'gt' | 'lt' | 'gte' | 'lte';

export type SmartAlertRule = {
  id: string;
  customer_id: string;
  campaign_id?: string;
  name: string;
  metric: SmartAlertMetric;
  operator: SmartAlertOperator;
  threshold: number;
  window_minutes: number;
  webhook_url: string;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
};

export type SmartAlertEvent = {
  id: string;
  rule_id: string;
  customer_id: string;
  campaign_id?: string;
  window_start: string;
  window_end: string;
  metric: string;
  operator: string;
  threshold: number;
  observed_value: number;
  webhook_status: string;
  webhook_error?: string;
  fired_at: string;
  acked_at?: string;
  acked_by?: string;
};

export type UpsertSmartAlertRuleBody = {
  customer_id: string;
  campaign_id?: string;
  name: string;
  metric: SmartAlertMetric;
  operator: SmartAlertOperator;
  threshold: number;
  window_minutes: number;
  webhook_url: string;
  enabled: boolean;
};

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

export async function fetchSmartAlertRules(customerId: string): Promise<SmartAlertRule[]> {
  const res = await api<SmartAlertRule[]>(
    `/api/v1/smart-alerts/rules?customer_id=${encodeURIComponent(customerId)}`,
  );
  return Array.isArray(res.data) ? res.data : [];
}

export async function createSmartAlertRule(body: UpsertSmartAlertRuleBody): Promise<SmartAlertRule> {
  const res = await apiConfirmed<SmartAlertRule>('/api/v1/smart-alerts/rules', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data;
}

export async function updateSmartAlertRule(
  id: string,
  body: Omit<UpsertSmartAlertRuleBody, 'customer_id'> & { customer_id?: string },
): Promise<SmartAlertRule> {
  const res = await apiConfirmed<SmartAlertRule>(
    `/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    },
  );
  return res.data;
}

export async function deleteSmartAlertRule(id: string): Promise<void> {
  await apiConfirmed(`/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function fetchSmartAlertHistory(
  customerId: string,
  limit = 50,
): Promise<SmartAlertEvent[]> {
  const res = await api<SmartAlertEvent[]>(
    `/api/v1/smart-alerts/history?customer_id=${encodeURIComponent(customerId)}&limit=${limit}`,
  );
  return Array.isArray(res.data) ? res.data : [];
}

export async function ackSmartAlertEvent(eventId: string): Promise<void> {
  await apiConfirmed(
    `/api/v1/smart-alerts/events/${encodeURIComponent(eventId)}/ack`,
    { method: 'POST' },
  );
}
