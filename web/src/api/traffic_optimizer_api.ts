import { apiJson } from './client.js';
import type {
  TrafficOptimizerDryRunResult,
  TrafficOptimizerListRulesQuery,
  TrafficOptimizerPreset,
  TrafficOptimizerRule,
  UpsertTrafficOptimizerRuleRequest,
} from './types.js';

export async function listTrafficOptimizerPresets(
  signal?: AbortSignal,
): Promise<TrafficOptimizerPreset[]> {
  return apiJson<TrafficOptimizerPreset[]>('/api/v1/traffic-optimizer/presets', { signal });
}

export async function listTrafficOptimizerRules(
  params: TrafficOptimizerListRulesQuery,
  signal?: AbortSignal,
): Promise<TrafficOptimizerRule[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  return apiJson<TrafficOptimizerRule[]>(
    `/api/v1/traffic-optimizer/rules?${search.toString()}`,
    { signal },
  );
}

export async function createTrafficOptimizerRule(
  body: UpsertTrafficOptimizerRuleRequest,
  signal?: AbortSignal,
): Promise<TrafficOptimizerRule> {
  return apiJson<TrafficOptimizerRule>('/api/v1/traffic-optimizer/rules', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateTrafficOptimizerRule(
  ruleId: string,
  body: UpsertTrafficOptimizerRuleRequest,
  signal?: AbortSignal,
): Promise<TrafficOptimizerRule> {
  return apiJson<TrafficOptimizerRule>(
    `/api/v1/traffic-optimizer/rules/${encodeURIComponent(ruleId)}`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function deleteTrafficOptimizerRule(
  ruleId: string,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<void>(`/api/v1/traffic-optimizer/rules/${encodeURIComponent(ruleId)}`, {
    method: 'DELETE',
    signal,
  });
}

export async function dryRunTrafficOptimizerRule(
  ruleId: string,
  signal?: AbortSignal,
): Promise<TrafficOptimizerDryRunResult> {
  return apiJson<TrafficOptimizerDryRunResult>(
    `/api/v1/traffic-optimizer/rules/${encodeURIComponent(ruleId)}/dry-run`,
    { method: 'POST', signal },
  );
}
