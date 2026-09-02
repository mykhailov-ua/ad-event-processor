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

test('active rows show red and yellow performance tones', () => {
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
    'admin-row-negative',
  );
  assert.equal(
    campaignListRowClass({
      status: 'ACTIVE',
      selected: false,
      margin: { operator_margin_micro: 100, margin_breach: true },
    }),
    'admin-row-warning',
  );
});

test('status row highlight pref no longer tints rows', () => {
  assert.equal(campaignListRowClass({ status: 'ACTIVE', selected: false }), '');
  assert.equal(
    campaignListRowClass({
      status: 'ACTIVE',
      selected: false,
      highlightActiveRows: true,
    }),
    '',
  );
});

test('paused rows still show performance tones', () => {
  assert.equal(
    campaignListRowClass({
      status: 'PAUSED',
      selected: false,
      margin: { operator_margin_micro: -500, margin_breach: true },
    }),
    'admin-row-negative',
  );
});

test('status dot is green only for active campaigns', () => {
  assert.equal(campaignListStatusDotClass('ACTIVE'), 'admin-table-status-dot--active');
  assert.equal(campaignListStatusDotClass('PAUSED'), 'admin-table-status-dot--muted');
  assert.equal(campaignListStatusDotClass('ARCHIVED'), 'admin-table-status-dot--muted');
});

test('status edge class maps lifecycle to left stripe', () => {
  assert.equal(campaignListRowStatusEdgeClass('ACTIVE'), 'admin-row-status-edge--active');
  assert.equal(campaignListRowStatusEdgeClass('PAUSED'), 'admin-row-status-edge--paused');
  assert.equal(campaignListRowStatusEdgeClass('ARCHIVED'), 'admin-row-status-edge--archived');
});
