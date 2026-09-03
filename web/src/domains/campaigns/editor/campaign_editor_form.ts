import type { Campaign, IngressCostConfig, PatchCampaignRequest } from '@/api/types';

import type { BuildCampaignPatchResult, CampaignEditorFormState } from './campaign_editor_types';

function ingressFromForm(form: CampaignEditorFormState): IngressCostConfig | undefined {
  const param = form.ingress_param.trim();
  if (param === '') {
    return undefined;
  }

  const config: IngressCostConfig = { param };
  const scale = form.ingress_scale.trim();
  if (scale !== '') {
    config.scale = scale;
  }
  const maxMicroText = form.ingress_max_micro.trim();
  if (maxMicroText !== '') {
    const parsed = Number(maxMicroText);
    if (!Number.isNaN(parsed)) {
      config.max_micro = parsed;
    }
  }
  const policy = form.ingress_policy.trim();
  if (policy !== '') {
    config.policy = policy;
  }
  return config;
}

function ingressConfigsEqual(
  left: IngressCostConfig | undefined,
  right: IngressCostConfig | undefined,
): boolean {
  return (
    (left?.param ?? '') === (right?.param ?? '') &&
    (left?.scale ?? '') === (right?.scale ?? '') &&
    String(left?.max_micro ?? '') === String(right?.max_micro ?? '') &&
    (left?.policy ?? '') === (right?.policy ?? '')
  );
}

function clickQueryParamsCanonicalJson(params: Record<string, string> | undefined): string {
  return JSON.stringify(params ?? {}, null, 2);
}

export function parseClickQueryParamsJson(
  json: string,
): { ok: true; value: Record<string, string> } | { ok: false; error: string } {
  const trimmed = json.trim();
  if (trimmed === '') {
    return { ok: true, value: {} };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, error: 'Click query params must be valid JSON.' };
  }

  if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return { ok: false, error: 'Click query params must be a JSON object.' };
  }

  const value: Record<string, string> = {};
  for (const [key, entry] of Object.entries(parsed as Record<string, unknown>)) {
    if (typeof entry !== 'string') {
      return {
        ok: false,
        error: `Click query params value for "${key}" must be a string.`,
      };
    }
    value[key] = entry;
  }
  return { ok: true, value };
}

function clickQueryParamsEqual(
  left: Record<string, string> | undefined,
  right: Record<string, string>,
): boolean {
  const leftObj = left ?? {};
  const leftKeys = Object.keys(leftObj).sort();
  const rightKeys = Object.keys(right).sort();
  if (leftKeys.length !== rightKeys.length) {
    return false;
  }
  return leftKeys.every((key, index) => key === rightKeys[index] && leftObj[key] === right[key]);
}

export function campaignToFormState(campaign: Campaign): CampaignEditorFormState {
  const ingress = campaign.ingress_cost_config;
  return {
    name: campaign.name,
    status: campaign.status,
    budget_limit: campaign.budget_limit,
    pacing_mode: campaign.pacing_mode,
    flow_id: campaign.flow_id ?? '',
    brand_id: campaign.brand_id ?? '',
    ingress_param: ingress?.param ?? '',
    ingress_scale: ingress?.scale ?? '',
    ingress_max_micro: ingress?.max_micro != null ? String(ingress.max_micro) : '',
    ingress_policy: ingress?.policy ?? '',
    traffic_template_id: campaign.traffic_template_id ?? '',
    click_query_params_json: clickQueryParamsCanonicalJson(campaign.click_query_params),
  };
}

export function buildCampaignPatchBody(
  original: Campaign,
  form: CampaignEditorFormState,
): BuildCampaignPatchResult {
  const body: PatchCampaignRequest = {};

  if (form.name !== original.name) {
    body.name = form.name;
  }
  if (form.status !== original.status) {
    body.status = form.status;
  }
  if (form.budget_limit !== original.budget_limit) {
    body.budget_limit = form.budget_limit;
  }
  if (form.pacing_mode !== original.pacing_mode) {
    body.pacing_mode = form.pacing_mode;
  }
  if (form.flow_id !== (original.flow_id ?? '')) {
    body.flow_id = form.flow_id;
  }
  if (form.brand_id !== (original.brand_id ?? '')) {
    body.brand_id = form.brand_id;
  }

  const nextIngress = ingressFromForm(form);
  if (!ingressConfigsEqual(nextIngress, original.ingress_cost_config)) {
    body.ingress_cost_config = nextIngress;
  }

  if (form.traffic_template_id !== (original.traffic_template_id ?? '')) {
    body.traffic_template_id = form.traffic_template_id;
  }

  const originalClickQueryJson = clickQueryParamsCanonicalJson(original.click_query_params);
  if (form.click_query_params_json !== originalClickQueryJson) {
    const parsed = parseClickQueryParamsJson(form.click_query_params_json);
    if (!parsed.ok) {
      return { ok: false, error: parsed.error };
    }
    if (!clickQueryParamsEqual(original.click_query_params, parsed.value)) {
      body.click_query_params = parsed.value;
    }
  }

  return { ok: true, body };
}
