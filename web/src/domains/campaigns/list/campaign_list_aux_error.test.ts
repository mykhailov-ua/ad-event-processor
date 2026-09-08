import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import {
  isCampaignListAuxEndpointUnavailable,
  isCampaignListAuxStaleControlRoute,
} from './campaign_list_aux_error.ts';

test('isCampaignListAuxStaleControlRoute detects shadowed {id} handler', () => {
  const err = new ApiError(400, 'INVALID_CAMPAIGN_ID', 'invalid campaign id');
  assert.equal(isCampaignListAuxStaleControlRoute(err), true);
  assert.equal(isCampaignListAuxEndpointUnavailable(err), true);
});

test('isCampaignListAuxStaleControlRoute ignores unrelated 400', () => {
  const err = new ApiError(400, 'INVALID_CUSTOMER_ID', 'invalid customer_id');
  assert.equal(isCampaignListAuxStaleControlRoute(err), false);
});
