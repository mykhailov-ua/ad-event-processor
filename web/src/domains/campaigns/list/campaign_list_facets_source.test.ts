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
  assert.equal(isCampaignListFacetsDegraded(undefined, true, undefined), false);
  assert.equal(isCampaignListFacetsDegraded(sampleFacets, true, undefined), false);
});

test('isCampaignListFacetsDegraded is false when list-facets API returns data', () => {
  assert.equal(isCampaignListFacetsDegraded(sampleFacets, false, undefined), false);
  assert.equal(isCampaignListFacetsDegraded({ countries: [], owners: [] }, false, undefined), false);
});

test('isCampaignListFacetsDegraded_holdout is true when fetch settled without API facets', () => {
  assert.equal(isCampaignListFacetsDegraded(undefined, false, undefined), true);
});

test('isCampaignListFacetsDegraded_holdout is false when fetch failed with error', () => {
  assert.equal(isCampaignListFacetsDegraded(undefined, false, new Error('INTERNAL_ERROR')), false);
});

test('resolveCampaignListFacets does not synthesize facets from list rows', () => {
  const resolved = resolveCampaignListFacets(undefined, false, undefined);
  assert.equal(resolved.degraded, true);
  assert.equal(resolved.facets, undefined);
});

test('resolveCampaignListFacets_holdout does not mark degraded on fetch error', () => {
  const resolved = resolveCampaignListFacets(undefined, false, new Error('INTERNAL_ERROR'));
  assert.equal(resolved.degraded, false);
  assert.equal(resolved.facets, undefined);
});
