import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type AutomationMetric =
  | 'clicks'
  | 'spend_micro'
  | 'roi_pct'
  | 'cr'
  | 'fraud_reject_count'
  | 'fraud_reject_rate'
  | 'ivt_rate'
  | 'silent_reject_rate';
export type AutomationOperator = 'gt' | 'lt' | 'gte' | 'lte';
export type AutomationGroupBy = 'placement_id' | 'campaign';
export type AutomationActionType =
  | 'notify'
  | 'pause_campaign'
  | 'blacklist_placement'
  | 'platform_pause';

export type AutomationAction = {
  type: AutomationActionType;
  webhook_url?: string;
  network?: string;
};

export type AutomationRule = {
  id: string;
  customer_id: string;
  campaign_id?: string;
  name: string;
  metric: AutomationMetric;
  operator: AutomationOperator;
  threshold: number;
  window_minutes: number;
  group_by: AutomationGroupBy;
  actions: AutomationAction[];
  cooldown_minutes: number;
  enabled: boolean;
  last_fired_at?: string;
  created_at?: string;
  updated_at?: string;
};

export type UpsertAutomationRuleBody = {
  customer_id: string;
  campaign_id?: string;
  name: string;
  metric: AutomationMetric;
  operator: AutomationOperator;
  threshold: number;
  window_minutes: number;
  group_by: AutomationGroupBy;
  actions: AutomationAction[];
  cooldown_minutes: number;
  enabled: boolean;
};

export const AUTOMATION_METRICS: AutomationMetric[] = [
  'roi_pct',
  'spend_micro',
  'clicks',
  'cr',
  'fraud_reject_rate',
  'ivt_rate',
  'silent_reject_rate',
  'fraud_reject_count',
];

export const AUTOMATION_METRIC_LABELS: Record<AutomationMetric, string> = {
  roi_pct: 'ROI (%)',
  spend_micro: 'Spend (micro)',
  clicks: 'Clicks',
  cr: 'CR (%)',
  fraud_reject_rate: 'Fraud reject rate (%)',
  ivt_rate: 'Fraud reject rate (%) - legacy key ivt_rate',
  silent_reject_rate: 'Silent reject rate (%)',
  fraud_reject_count: 'Fraud reject count',
};
export const AUTOMATION_OPERATORS: AutomationOperator[] = ['lt', 'lte', 'gt', 'gte'];
export const AUTOMATION_ACTION_TYPES: AutomationActionType[] = [
  'pause_campaign',
  'blacklist_placement',
  'notify',
  'platform_pause',
];

/** Fetch automation rules for a customer. */
export async function fetchAutomationRules(customerId: string): Promise<AutomationRule[]> {
  const { data } = await api(
    `/api/v1/automation/rules?customer_id=${encodeURIComponent(customerId)}`
  );
  return (data as AutomationRule[] | null | undefined) ?? [];
}

/** Create an automation rule. */
export async function createAutomationRule(
  body: UpsertAutomationRuleBody
): Promise<AutomationRule> {
  const res = await apiConfirmed('/api/v1/automation/rules', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data as AutomationRule;
}

/** Update an automation rule. */
export async function updateAutomationRule(
  ruleId: string,
  body: UpsertAutomationRuleBody
): Promise<AutomationRule> {
  const res = await apiConfirmed(`/api/v1/automation/rules/${encodeURIComponent(ruleId)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data as AutomationRule;
}

/** Delete an automation rule. */
export async function deleteAutomationRule(ruleId: string): Promise<void> {
  await apiConfirmed(`/api/v1/automation/rules/${encodeURIComponent(ruleId)}`, {
    method: 'DELETE',
  });
}

/** Dry-run a rule against current ClickHouse rollups. */
export async function dryRunAutomationRule(ruleId: string): Promise<{ would_fire: unknown[] }> {
  const res = await apiConfirmed(`/api/v1/automation/rules/${encodeURIComponent(ruleId)}/dry-run`, {
    method: 'POST',
    body: JSON.stringify({}),
  });
  return res.data as { would_fire: unknown[] };
}
