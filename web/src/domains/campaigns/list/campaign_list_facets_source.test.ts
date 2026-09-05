import assert from 'node:assert/strict';
import test from 'node:test';

import {
  isCampaignListFacetsDegraded,
  resolveCampaignListFacets,
} from './campaign_list_facets_source.ts';

const sampleFacets = {
  countries: ['US'],
  owners: [{ user_id: '11111111-1111-4111-8111-111111111101' }],
};

test('isCampaignListFacetsDegraded is false while facets fetch is in flight', () => {
  assert.equal(isCampaignListFacetsDegraded(undefined, true), false);
  assert.equal(isCampaignListFacetsDegraded(sampleFacets, true), false);
});

test('isCampaignListFacetsDegraded is false when list-facets API returns data', () => {
  assert.equal(isCampaignListFacetsDegraded(sampleFacets, false), false);
  assert.equal(isCampaignListFacetsDegraded({ countries: [], owners: [] }, false), false);
});

test('isCampaignListFacetsDegraded_holdout is true when fetch settled without API facets', () => {
  assert.equal(isCampaignListFacetsDegraded(undefined, false), true);
});

test('resolveCampaignListFacets does not synthesize facets from list rows', () => {
  const resolved = resolveCampaignListFacets(undefined, false);
  assert.equal(resolved.degraded, true);
  assert.equal(resolved.facets, undefined);
});
