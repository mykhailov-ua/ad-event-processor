import type {
  AutomationRule,
  SmartAlertRule,
  TrafficOptimizerRule,
  UpsertAutomationRuleRequest,
  UpsertSmartAlertRuleRequest,
  UpsertTrafficOptimizerRuleRequest,
} from '@/api/types';

export type AutomationRuleEditDraft = {
  name: string;
  metric: string;
  operator: string;
  threshold: string;
  enabled: boolean;
};

export type TrafficOptimizerRuleEditDraft = {
  name: string;
  enabled: boolean;
};

export type SmartAlertRuleEditDraft = {
  name: string;
  metric: string;
  operator: string;
  threshold: string;
  window_minutes: string;
  webhook_url: string;
  enabled: boolean;
};

export function automationRuleEditFromRow(row: AutomationRule): AutomationRuleEditDraft {
  return {
    name: row.name ?? '',
    metric: row.metric ?? '',
    operator: row.operator ?? '',
    threshold: row.threshold != null ? String(row.threshold) : '',
    enabled: row.enabled ?? false,
  };
}

export function automationRuleUpsertBody(
  customerId: string,
  row: AutomationRule,
  draft: AutomationRuleEditDraft,
): UpsertAutomationRuleRequest {
  const threshold = Number.parseFloat(draft.threshold.trim());
  return {
    customer_id: customerId,
    campaign_id: row.campaign_id,
    name: draft.name.trim(),
    metric: draft.metric.trim(),
    operator: draft.operator.trim(),
    threshold: Number.isFinite(threshold) ? threshold : row.threshold ?? 0,
    window_minutes: row.window_minutes,
    group_by: row.group_by,
    actions: row.actions ?? [],
    cooldown_minutes: row.cooldown_minutes,
    eval_interval_minutes: row.eval_interval_minutes,
    enabled: draft.enabled,
  };
}

export function automationRuleCreateBody(
  customerId: string,
  draft: AutomationRuleEditDraft,
): UpsertAutomationRuleRequest {
  const threshold = Number.parseFloat(draft.threshold.trim());
  return {
    customer_id: customerId,
    enabled: draft.enabled,
    name: draft.name.trim(),
    metric: draft.metric.trim() || 'spend_micro',
    operator: draft.operator.trim() || 'gt',
    threshold: Number.isFinite(threshold) ? threshold : 0,
    window_minutes: 60,
    group_by: 'placement',
    actions: [{ type: 'notify', webhook_url: 'https://hooks.example.invalid/alert' }],
    cooldown_minutes: 60,
    eval_interval_minutes: 15,
  };
}

export function trafficOptimizerRuleEditFromRow(row: TrafficOptimizerRule): TrafficOptimizerRuleEditDraft {
  return {
    name: row.name ?? '',
    enabled: row.enabled ?? false,
  };
}

export function trafficOptimizerRuleUpsertBody(
  customerId: string,
  row: TrafficOptimizerRule,
  draft: TrafficOptimizerRuleEditDraft,
): UpsertTrafficOptimizerRuleRequest {
  return {
    customer_id: customerId,
    campaign_id: row.campaign_id,
    flow_id: row.flow_id,
    brand_id: row.brand_id,
    name: draft.name.trim(),
    scope: row.scope,
    objective: row.objective,
    algorithm: row.algorithm,
    lookback_minutes: row.lookback_minutes,
    min_clicks: row.min_clicks,
    min_conversions: row.min_conversions,
    min_spend_micro: row.min_spend_micro,
    eval_interval_minutes: row.eval_interval_minutes,
    cooldown_minutes: row.cooldown_minutes,
    max_weight_delta_pct: row.max_weight_delta_pct,
    preset_key: row.preset_key,
    enabled: draft.enabled,
  };
}

export function trafficOptimizerRuleCreateBody(
  customerId: string,
  draft: TrafficOptimizerRuleEditDraft,
): UpsertTrafficOptimizerRuleRequest {
  return {
    customer_id: customerId,
    enabled: draft.enabled,
    name: draft.name.trim(),
    scope: 'lander',
    objective: 'cr',
    algorithm: 'thompson',
    lookback_minutes: 60,
    min_clicks: 10,
    min_conversions: 1,
    min_spend_micro: 1000000,
    eval_interval_minutes: 15,
    cooldown_minutes: 60,
    max_weight_delta_pct: 25,
  };
}

export function smartAlertRuleEditFromRow(row: SmartAlertRule): SmartAlertRuleEditDraft {
  return {
    name: row.name ?? '',
    metric: row.metric ?? '',
    operator: row.operator ?? '',
    threshold: row.threshold != null ? String(row.threshold) : '',
    window_minutes: row.window_minutes != null ? String(row.window_minutes) : '',
    webhook_url: row.webhook_url ?? '',
    enabled: row.enabled ?? false,
  };
}

export function smartAlertRuleUpsertBody(
  customerId: string,
  row: SmartAlertRule,
  draft: SmartAlertRuleEditDraft,
): UpsertSmartAlertRuleRequest {
  const threshold = Number.parseFloat(draft.threshold.trim());
  const windowMinutes = Number.parseInt(draft.window_minutes.trim(), 10);
  return {
    customer_id: customerId,
    campaign_id: row.campaign_id,
    name: draft.name.trim(),
    metric: draft.metric.trim(),
    operator: draft.operator.trim(),
    threshold: Number.isFinite(threshold) ? threshold : row.threshold ?? 0,
    window_minutes: Number.isFinite(windowMinutes) ? windowMinutes : row.window_minutes ?? 60,
    webhook_url: draft.webhook_url.trim(),
    enabled: draft.enabled,
  };
}

export function smartAlertRuleCreateBody(
  customerId: string,
  draft: SmartAlertRuleEditDraft,
): UpsertSmartAlertRuleRequest {
  const threshold = Number.parseFloat(draft.threshold.trim());
  const windowMinutes = Number.parseInt(draft.window_minutes.trim(), 10);
  return {
    customer_id: customerId,
    enabled: draft.enabled,
    name: draft.name.trim(),
    metric: draft.metric.trim() || 'spend_micro',
    operator: draft.operator.trim() || 'gt',
    threshold: Number.isFinite(threshold) ? threshold : 0,
    window_minutes: Number.isFinite(windowMinutes) ? windowMinutes : 60,
    webhook_url: draft.webhook_url.trim() || 'https://hooks.example.invalid/alert',
  };
}
