import { apiFetch, apiJsonValidated, parseApiError } from './client.js';
import {
  parseTrafficOptimizerDryRunResult,
  parseTrafficOptimizerPresetList,
  parseTrafficOptimizerRule,
  parseTrafficOptimizerRuleList,
} from './validate.js';
import type {
  TrafficOptimizerDryRunResult,
  TrafficOptimizerListRulesQuery,
  TrafficOptimizerPreset,
  TrafficOptimizerRule,
  UpsertTrafficOptimizerRuleRequest,
} from './types.js';

export async function listTrafficOptimizerPresets(
  signal?: AbortSignal
): Promise<TrafficOptimizerPreset[]> {
  return apiJsonValidated(
    '/api/v1/traffic-optimizer/presets',
    { signal },
    parseTrafficOptimizerPresetList
  );
}

export async function listTrafficOptimizerRules(
  params: TrafficOptimizerListRulesQuery,
  signal?: AbortSignal
): Promise<TrafficOptimizerRule[]> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  const query = search.toString();
  const path = query
    ? `/api/v1/traffic-optimizer/rules?${query}`
    : '/api/v1/traffic-optimizer/rules';
  return apiJsonValidated(path, { signal }, parseTrafficOptimizerRuleList);
}

export async function createTrafficOptimizerRule(
  body: UpsertTrafficOptimizerRuleRequest,
  signal?: AbortSignal
): Promise<TrafficOptimizerRule> {
  return apiJsonValidated(
    '/api/v1/traffic-optimizer/rules',
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
    parseTrafficOptimizerRule
  );
}

export async function deleteTrafficOptimizerRule(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/traffic-optimizer/rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
}

export async function dryRunTrafficOptimizerRule(
  id: string,
  signal?: AbortSignal
): Promise<TrafficOptimizerDryRunResult> {
  return apiJsonValidated(
    `/api/v1/traffic-optimizer/rules/${encodeURIComponent(id)}/dry-run`,
    {
      method: 'POST',
      signal,
    },
    parseTrafficOptimizerDryRunResult
  );
}

export type TrafficOptimizerApplyResult = {
  applied: boolean;
  campaign_ids?: string[];
};

function parseTrafficOptimizerApplyResult(value: unknown): TrafficOptimizerApplyResult {
  if (
    value == null ||
    typeof value !== 'object' ||
    typeof (value as TrafficOptimizerApplyResult).applied !== 'boolean'
  ) {
    throw new Error('Traffic optimizer apply result missing applied flag');
  }
  return value as TrafficOptimizerApplyResult;
}

export async function applyTrafficOptimizerRule(
  id: string,
  signal?: AbortSignal
): Promise<TrafficOptimizerApplyResult> {
  return apiJsonValidated(
    `/api/v1/traffic-optimizer/rules/${encodeURIComponent(id)}/apply`,
    {
      method: 'POST',
      signal,
    },
    parseTrafficOptimizerApplyResult
  );
}
