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
  assert.ok(campaignListRowClass(false).includes('odd:[&_td]:bg-background'));
  assert.equal(campaignListRowClass(false).includes('[&_td]:bg-accent/80'), false);
});

test('selected row uses highlight class', () => {
  assert.ok(campaignListRowClass(true).includes('[&_td]:bg-accent/80'));
});

test('campaignStatusBadgeClass tints active campaigns green', () => {
  assert.match(campaignStatusBadgeClass('ACTIVE'), /text-emerald-500/);
  assert.match(campaignStatusBadgeClass('PAUSED'), /text-amber-500/);
  assert.match(campaignStatusBadgeClass('ACTIVE'), /rounded-full/);
});
