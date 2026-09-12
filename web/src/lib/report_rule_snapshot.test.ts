import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildSourceQualityFilterSnapshot,
  defaultReportRuleActionForRow,
  resolveReportRuleCampaignId,
} from '@/lib/report_rule_snapshot';

test('defaultReportRuleActionForRow prefers blacklist when placement present', () => {
  assert.equal(defaultReportRuleActionForRow({ placement_id: 'slot-1' }), 'blacklist_placement');
  assert.equal(
    defaultReportRuleActionForRow({ campaign_id: '00000000-0000-4000-8000-000000000010' }),
    'pause_campaign'
  );
});

test('buildSourceQualityFilterSnapshot captures row and filter context', () => {
  const snapshot = buildSourceQualityFilterSnapshot(
    { placement_id: 'slot-1', country: 'US' },
    {
      customerId: '00000000-0000-4000-8000-000000000001',
      campaignId: '00000000-0000-4000-8000-000000000010',
      from: '2026-09-01T00:00:00Z',
      to: '2026-09-12T00:00:00Z',
      groupBy: ['placement', 'country'],
      compare: true,
    }
  );
  assert.equal(snapshot.placement_id, 'slot-1');
  assert.equal(snapshot.country, 'US');
  assert.equal(snapshot.campaign_id, '00000000-0000-4000-8000-000000000010');
  assert.equal(snapshot.group_by, 'placement,country');
  assert.equal(snapshot.compare, '1');
});

test('resolveReportRuleCampaignId prefers row campaign', () => {
  assert.equal(
    resolveReportRuleCampaignId(
      { campaign_id: '00000000-0000-4000-8000-000000000099' },
      { customerId: 'x', campaignId: '00000000-0000-4000-8000-000000000010' }
    ),
    '00000000-0000-4000-8000-000000000099'
  );
});
