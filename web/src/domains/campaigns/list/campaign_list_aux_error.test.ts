import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import {
  isCampaignListAuxEndpointUnavailable,
  isCampaignListAuxStaleControlRoute,
} from './campaign_list_aux_error.ts';

test('isCampaignListAuxStaleControlRoute detects shadowed {id} handler', () => {
  const err = new ApiError('invalid campaign id', 400);
  assert.equal(isCampaignListAuxStaleControlRoute(err), true);
  assert.equal(isCampaignListAuxEndpointUnavailable(err), true);
});

test('isCampaignListAuxStaleControlRoute ignores unrelated 400', () => {
  const err = new ApiError('invalid customer_id', 400);
  assert.equal(isCampaignListAuxStaleControlRoute(err), false);
});
