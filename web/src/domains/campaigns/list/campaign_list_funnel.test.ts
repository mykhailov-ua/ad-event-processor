import assert from 'node:assert/strict';
import test from 'node:test';

import {
  resolveCampaignFunnelCounts,
  syncCampaignFunnelLeadsRaw,
} from './campaign_list_funnel.ts';

test('syncCampaignFunnelLeadsRaw keeps explicit leads_raw', () => {
  assert.equal(syncCampaignFunnelLeadsRaw(3, 1, 2, 99), 99);
});

test('syncCampaignFunnelLeadsRaw derives from funnel parts', () => {
  assert.equal(syncCampaignFunnelLeadsRaw(3, 1, 2, undefined), 6);
});

test('syncCampaignFunnelLeadsRaw_holdout falls back to approved when funnel empty', () => {
  assert.equal(syncCampaignFunnelLeadsRaw(4, 0, 0, 0), 4);
});

test('resolveCampaignFunnelCounts uses API leads_raw when present', () => {
  const funnel = resolveCampaignFunnelCounts({
    conversions: 3,
    hold_leads: 1,
    rejected_leads: 2,
    leads_raw: 20,
  });
  assert.equal(funnel.rawLeads, 20);
});

test('resolveCampaignFunnelCounts_holdout derives leads when leads_raw missing', () => {
  const funnel = resolveCampaignFunnelCounts({
    conversions: 3,
    hold_leads: 1,
    rejected_leads: 2,
  });
  assert.equal(funnel.rawLeads, 6);
});
