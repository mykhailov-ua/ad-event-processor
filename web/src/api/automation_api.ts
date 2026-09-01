import { apiJson } from './client.js';
import type {
  AutomationDryRunResult,
  AutomationListRulesQuery,
  AutomationPreset,
  AutomationRule,
  UpsertAutomationRuleRequest,
} from './types.js';

export async function listAutomationPresets(signal?: AbortSignal): Promise<AutomationPreset[]> {
  return apiJson<AutomationPreset[]>('/api/v1/automation/presets', { signal });
}

export async function listAutomationRules(
  params: AutomationListRulesQuery,
  signal?: AbortSignal,
): Promise<AutomationRule[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  return apiJson<AutomationRule[]>(`/api/v1/automation/rules?${search.toString()}`, { signal });
}

export async function createAutomationRule(
  body: UpsertAutomationRuleRequest,
  signal?: AbortSignal,
): Promise<AutomationRule> {
  return apiJson<AutomationRule>('/api/v1/automation/rules', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateAutomationRule(
  ruleId: string,
  body: UpsertAutomationRuleRequest,
  signal?: AbortSignal,
): Promise<AutomationRule> {
  return apiJson<AutomationRule>(`/api/v1/automation/rules/${encodeURIComponent(ruleId)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteAutomationRule(ruleId: string, signal?: AbortSignal): Promise<void> {
  await apiJson<void>(`/api/v1/automation/rules/${encodeURIComponent(ruleId)}`, {
    method: 'DELETE',
    signal,
  });
}

export async function dryRunAutomationRule(
  ruleId: string,
  signal?: AbortSignal,
): Promise<AutomationDryRunResult> {
  return apiJson<AutomationDryRunResult>(
    `/api/v1/automation/rules/${encodeURIComponent(ruleId)}/dry-run`,
    { method: 'POST', signal },
  );
}
