import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignListRowClass,
  campaignStatusBadgeClass,
  isInactiveCampaignStatus,
  normalizeCampaignStatus,
  resolveCampaignStatusKey,
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
  assert.equal(campaignListRowClass(false), '');
  assert.equal(campaignListRowClass(false), '');
});

test('selected row uses highlight class', () => {
  assert.equal(campaignListRowClass(true), 'bg-blue-50 dark:bg-blue-950/30');
});

test('campaignStatusBadgeClass tints active campaigns green', () => {
  assert.match(campaignStatusBadgeClass('ACTIVE'), /text-green-700/);
  assert.match(campaignStatusBadgeClass('PAUSED'), /text-amber-700/);
});
