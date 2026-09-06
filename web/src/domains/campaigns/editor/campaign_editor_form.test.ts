import test from 'node:test';
import assert from 'node:assert/strict';

import type { Campaign } from '@/api/types';
import { CLEAR_UUID } from '@/lib/clear_uuid';

import {
  buildCampaignPatchBody,
  campaignToFormState,
  parseClickQueryParamsJson,
} from './campaign_editor_form.ts';
import {
  CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS,
  CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN,
  CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT,
  CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS,
} from './campaign_click_query_limits.ts';

test('click query params field hint documents whole-json and per-value limits', () => {
  assert.match(CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT, /21,442/);
  assert.match(
    CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT,
    new RegExp(String(CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN))
  );
  assert.match(
    CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT,
    new RegExp(String(CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS))
  );
  assert.equal(CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS, 21442);
});

function baseCampaign(overrides: Partial<Campaign> = {}): Campaign {
  return {
    id: 'cmp-1',
    name: 'Test campaign',
    status: 'PAUSED',
    budget_limit: '1000000',
    pacing_mode: 'even',
    click_query_params: { sub1: 'a' },
    ...overrides,
  };
}

test('parseClickQueryParamsJson accepts empty input as empty object', () => {
  assert.deepEqual(parseClickQueryParamsJson(''), { ok: true, value: {} });
});

test('parseClickQueryParamsJson rejects non-object JSON', () => {
  assert.equal(parseClickQueryParamsJson('[]').ok, false);
});

test('parseClickQueryParamsJson rejects non-string values', () => {
  const result = parseClickQueryParamsJson('{"sub1":1}');
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.match(result.error, /sub1/);
  }
});

test('campaignToFormState maps ingress and click query params with sorted keys', () => {
  const form = campaignToFormState(
    baseCampaign({
      ingress_cost_config: {
        param: 'cost',
        scale: '1.5',
        max_micro: 2500,
        policy: 'cap',
      },
      click_query_params: { sub1: 'x', country: 'US' },
    })
  );

  assert.equal(form.ingress_param, 'cost');
  assert.equal(form.ingress_scale, '1.5');
  assert.equal(form.ingress_max_micro, '2500');
  assert.equal(form.ingress_policy, 'cap');
  assert.equal(form.click_query_params_json, JSON.stringify({ country: 'US', sub1: 'x' }, null, 2));
});

test('buildCampaignPatchBody returns empty body when form matches snapshot', () => {
  const campaign = baseCampaign();
  const form = campaignToFormState(campaign);
  const result = buildCampaignPatchBody(campaign, form);
  assert.equal(result.ok, true);
  if (result.ok) {
    assert.deepEqual(result.body, {});
  }
});

test('buildCampaignPatchBody includes changed scalar fields', () => {
  const campaign = baseCampaign();
  const form = campaignToFormState(campaign);
  form.name = 'Renamed';
  form.status = 'ACTIVE';

  const result = buildCampaignPatchBody(campaign, form);
  assert.deepEqual(result, {
    ok: true,
    body: { name: 'Renamed', status: 'ACTIVE' },
  });
});

test('buildCampaignPatchBody fails when click query params JSON is invalid', () => {
  const campaign = baseCampaign();
  const form = campaignToFormState(campaign);
  form.click_query_params_json = '{bad';

  assert.equal(buildCampaignPatchBody(campaign, form).ok, false);
});

test('buildCampaignPatchBody_holdoutClearsFlowIdWithNilUuid', () => {
  const campaign = baseCampaign({
    flow_id: '11111111-1111-4111-8111-111111111111',
  });
  const form = campaignToFormState(campaign);
  form.flow_id = '';

  const result = buildCampaignPatchBody(campaign, form);
  assert.deepEqual(result, {
    ok: true,
    body: { flow_id: CLEAR_UUID },
  });
});

test('buildCampaignPatchBody_holdoutRejectsInvalidIngressMaxMicro', () => {
  const campaign = baseCampaign({
    ingress_cost_config: { param: 'cost', max_micro: 100 },
  });
  const form = campaignToFormState(campaign);
  form.ingress_max_micro = 'not-a-number';

  const result = buildCampaignPatchBody(campaign, form);
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.match(result.error, /max micro/i);
  }
});

test('parseClickQueryParamsJson rejects JSON longer than server bound', () => {
  const result = parseClickQueryParamsJson(
    `{${' '.repeat(CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS + 1)}}`
  );
  assert.equal(result.ok, false);
});

test('parseClickQueryParamsJson rejects too many keys', () => {
  const params: Record<string, string> = {};
  for (let i = 1; i <= CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS + 1; i += 1) {
    params[`sub${i}`] = 'x';
  }
  const result = parseClickQueryParamsJson(JSON.stringify(params));
  assert.equal(result.ok, false);
});

test('parseClickQueryParamsJson rejects value longer than server bound', () => {
  const result = parseClickQueryParamsJson(
    JSON.stringify({ sub1: 'a'.repeat(CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN + 1) })
  );
  assert.equal(result.ok, false);
});

test('buildCampaignPatchBody_holdoutIgnoresClickQueryKeyReorder', () => {
  const campaign = baseCampaign({
    click_query_params: { b: '2', a: '1' },
  });
  const form = campaignToFormState(campaign);
  form.click_query_params_json = JSON.stringify({ a: '1', b: '2' }, null, 2);

  const result = buildCampaignPatchBody(campaign, form);
  assert.equal(result.ok, true);
  if (result.ok) {
    assert.deepEqual(result.body, {});
  }
});

test('buildCampaignPatchBody_holdoutDetectsMaxMicroZeroCleared', () => {
  const campaign = baseCampaign({
    ingress_cost_config: { param: 'cost', max_micro: 0 },
  });
  const form = campaignToFormState(campaign);
  form.ingress_max_micro = '';

  const result = buildCampaignPatchBody(campaign, form);
  assert.equal(result.ok, true);
  if (result.ok) {
    assert.deepEqual(result.body.ingress_cost_config, { param: 'cost' });
  }
});
