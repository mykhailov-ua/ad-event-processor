import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignListRowClass,
  campaignListRowStatusEdgeClass,
  campaignListStatusDotClass,
  isInactiveCampaignStatus,
  normalizeCampaignStatus,
  resolveCampaignStatusKey,
  resolvePerformanceRowTone,
} from './campaign_list_row_tone.ts';

test('normalizeCampaignStatus maps known statuses', () => {
  assert.equal(normalizeCampaignStatus('active'), 'ACTIVE');
  assert.equal(normalizeCampaignStatus(' Paused '), 'PAUSED');
  assert.equal(normalizeCampaignStatus('draft'), 'UNKNOWN');
});

test('resolveCampaignStatusKey prefers status_tone', () => {
  assert.equal(resolveCampaignStatusKey('ACTIVE', 'warning'), 'PAUSED');
  assert.equal(resolveCampaignStatusKey('ARCHIVED', 'success'), 'ACTIVE');
});

test('inactive statuses do not tint table rows', () => {
  assert.equal(isInactiveCampaignStatus('PAUSED'), true);
  assert.equal(campaignListRowClass({ status: 'PAUSED', selected: false }), '');
  assert.equal(campaignListRowClass({ status: 'ARCHIVED', selected: false }), '');
});

test('active rows no longer use row background tones', () => {
  assert.equal(
    resolvePerformanceRowTone({ operator_margin_micro: -100, margin_breach: true }),
    'negative',
  );
  assert.equal(
    campaignListRowClass({
      status: 'ACTIVE',
      selected: false,
      margin: { operator_margin_micro: -100, margin_breach: true },
    }),
    '',
  );
  assert.equal(
    campaignListRowClass({
      status: 'ACTIVE',
      selected: false,
      margin: { operator_margin_micro: 100, margin_breach: true },
    }),
    '',
  );
});

test('selected row uses highlight class', () => {
  assert.equal(campaignListRowClass({ status: 'ACTIVE', selected: true }), 'bg-blue-50 dark:bg-blue-950/30');
});

test('paused rows no longer use row background tones', () => {
  assert.equal(
    campaignListRowClass({
      status: 'PAUSED',
      selected: false,
      margin: { operator_margin_micro: -500, margin_breach: true },
    }),
    '',
  );
});

test('status dot is green only for active campaigns', () => {
  assert.equal(campaignListStatusDotClass('ACTIVE'), 'inline-block h-2 w-2 rounded-full bg-green-500');
  assert.equal(campaignListStatusDotClass('PAUSED'), 'inline-block h-2 w-2 rounded-full bg-zinc-400');
  assert.equal(campaignListStatusDotClass('ARCHIVED'), 'inline-block h-2 w-2 rounded-full bg-zinc-400');
});

test('status edge class is unused (empty)', () => {
  assert.equal(campaignListRowStatusEdgeClass('ACTIVE'), '');
  assert.equal(campaignListRowStatusEdgeClass('PAUSED'), '');
  assert.equal(campaignListRowStatusEdgeClass('ARCHIVED'), '');
});
