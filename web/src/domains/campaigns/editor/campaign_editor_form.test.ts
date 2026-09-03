import test from 'node:test';
import assert from 'node:assert/strict';

import type { Campaign } from '@/api/types';

import {
  buildCampaignPatchBody,
  campaignToFormState,
  parseClickQueryParamsJson,
} from './campaign_editor_form.ts';

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

test('campaignToFormState maps ingress and click query params', () => {
  const form = campaignToFormState(
    baseCampaign({
      ingress_cost_config: {
        param: 'cost',
        scale: '1.5',
        max_micro: 2500,
        policy: 'cap',
      },
      click_query_params: { sub1: 'x', country: 'US' },
    }),
  );

  assert.equal(form.ingress_param, 'cost');
  assert.equal(form.ingress_scale, '1.5');
  assert.equal(form.ingress_max_micro, '2500');
  assert.equal(form.ingress_policy, 'cap');
  assert.equal(
    form.click_query_params_json,
    JSON.stringify({ sub1: 'x', country: 'US' }, null, 2),
  );
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
