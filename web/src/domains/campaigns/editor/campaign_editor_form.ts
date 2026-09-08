import type { Campaign, IngressCostConfig, PatchCampaignRequest } from '@/api/types';
import { resolveOptionalUuidPatchValue } from '@/lib/clear_uuid';

import type { BuildCampaignPatchResult, CampaignEditorFormState } from './campaign_editor_types';
import {
  CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS,
  CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN,
  CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS,
} from './campaign_click_query_limits';

type IngressFromFormResult =
  | { ok: true; value: IngressCostConfig | undefined }
  | { ok: false; error: string };

function ingressFromForm(form: CampaignEditorFormState): IngressFromFormResult {
  const param = form.ingress_param.trim();
  if (param === '') {
    return { ok: true, value: undefined };
  }

  const config: IngressCostConfig = { param };
  const scale = form.ingress_scale.trim();
  if (scale !== '') {
    config.scale = scale;
  }
  const maxMicroText = form.ingress_max_micro.trim();
  if (maxMicroText !== '') {
    const parsed = Number(maxMicroText);
    if (Number.isNaN(parsed)) {
      return { ok: false, error: 'Ingress max micro must be a valid number.' };
    }
    config.max_micro = parsed;
  }
  const policy = form.ingress_policy.trim();
  if (policy !== '') {
    config.policy = policy;
  }
  return { ok: true, value: config };
}

function ingressConfigsEqual(
  left: IngressCostConfig | undefined,
  right: IngressCostConfig | undefined
): boolean {
  return (
    (left?.param ?? '') === (right?.param ?? '') &&
    (left?.scale ?? '') === (right?.scale ?? '') &&
    (left?.max_micro ?? undefined) === (right?.max_micro ?? undefined) &&
    (left?.policy ?? '') === (right?.policy ?? '')
  );
}

function clickQueryParamsCanonicalJson(params: Record<string, string> | undefined): string {
  const sorted: Record<string, string> = {};
  for (const key of Object.keys(params ?? {}).sort()) {
    sorted[key] = params![key];
  }
  return JSON.stringify(sorted, null, 2);
}

export function parseClickQueryParamsJson(
  json: string
): { ok: true; value: Record<string, string> } | { ok: false; error: string } {
  const trimmed = json.trim();
  if (trimmed === '') {
    return { ok: true, value: {} };
  }
  if (trimmed.length > CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS) {
    return {
      ok: false,
      error: `Click query params JSON must be at most ${CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS.toLocaleString()} characters.`,
    };
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
  if (Object.keys(value).length > CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS) {
    return {
      ok: false,
      error: `Click query params allow at most ${CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS} keys.`,
    };
  }
  for (const [key, entry] of Object.entries(value)) {
    if (entry.length > CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN) {
      return {
        ok: false,
        error: `Click query params value for "${key}" exceeds ${CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN} characters.`,
      };
    }
  }
  return { ok: true, value };
}

function clickQueryParamsEqual(
  left: Record<string, string> | undefined,
  right: Record<string, string>
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
    click_filter_tier: campaign.click_filter_tier ?? 'full',
    mobile_biometrics_click_enabled: campaign.mobile_biometrics_click_enabled ?? false,
    decoy_lander_id: campaign.decoy_lander_id ?? '',
    redirect_compliance_mode: campaign.redirect_compliance_mode ?? 'strict',
    dmr_enabled: campaign.dmr_enabled ?? false,
  };
}

export function buildCampaignPatchBody(
  original: Campaign,
  form: CampaignEditorFormState
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

  const nextFlowId = resolveOptionalUuidPatchValue(form.flow_id, original.flow_id);
  if (nextFlowId !== undefined) {
    body.flow_id = nextFlowId;
  }

  const nextBrandId = resolveOptionalUuidPatchValue(form.brand_id, original.brand_id);
  if (nextBrandId !== undefined) {
    body.brand_id = nextBrandId;
  }

  const ingressResult = ingressFromForm(form);
  if (!ingressResult.ok) {
    return ingressResult;
  }
  if (!ingressConfigsEqual(ingressResult.value, original.ingress_cost_config)) {
    body.ingress_cost_config = ingressResult.value;
  }

  if (form.traffic_template_id !== (original.traffic_template_id ?? '')) {
    body.traffic_template_id = form.traffic_template_id;
  }

  const parsedClickQuery = parseClickQueryParamsJson(form.click_query_params_json);
  if (!parsedClickQuery.ok) {
    return { ok: false, error: parsedClickQuery.error };
  }
  if (!clickQueryParamsEqual(original.click_query_params, parsedClickQuery.value)) {
    body.click_query_params = parsedClickQuery.value;
  }

  const originalTier = original.click_filter_tier ?? 'full';
  if (form.click_filter_tier !== originalTier) {
    body.click_filter_tier = form.click_filter_tier as PatchCampaignRequest['click_filter_tier'];
  }

  if (
    form.mobile_biometrics_click_enabled !== (original.mobile_biometrics_click_enabled ?? false)
  ) {
    body.mobile_biometrics_click_enabled = form.mobile_biometrics_click_enabled;
  }

  const nextDecoyLanderID = resolveOptionalUuidPatchValue(
    form.decoy_lander_id,
    original.decoy_lander_id
  );
  if (nextDecoyLanderID !== undefined) {
    body.decoy_lander_id = nextDecoyLanderID;
  }

  const originalRedirectMode = original.redirect_compliance_mode ?? 'strict';
  if (form.redirect_compliance_mode !== originalRedirectMode) {
    body.redirect_compliance_mode =
      form.redirect_compliance_mode as PatchCampaignRequest['redirect_compliance_mode'];
  }

  if (form.dmr_enabled !== (original.dmr_enabled ?? false)) {
    body.dmr_enabled = form.dmr_enabled;
  }

  return { ok: true, body };
}
