import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignListRowDataAttributes,
  campaignStatusBadgeClass,
  campaignStatusCellClass,
  isInactiveCampaignStatus,
  normalizeCampaignStatus,
  resolveCampaignListRowAccent,
  resolveCampaignListRowAlert,
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
  assert.deepEqual(campaignListRowDataAttributes(false), {});
});

test('campaignListRowDataAttributes_holdout: selection wins over accent', () => {
  assert.deepEqual(campaignListRowDataAttributes(true, 'warning'), { 'data-row-selected': true });
  assert.equal(campaignListRowDataAttributes(true, 'warning')['data-row-accent'], undefined);
});

test('resolveCampaignListRowAccent_holdout does not tint rows; alerts use name badge only', () => {
  assert.equal(resolveCampaignListRowAccent('PAUSED'), 'none');
  assert.equal(resolveCampaignListRowAccent('ARCHIVED'), 'none');
  assert.equal(resolveCampaignListRowAccent('ACTIVE'), 'none');
});

test('resolveCampaignListRowAlert maps problem signals for name column badge', () => {
  assert.equal(
    resolveCampaignListRowAlert('ACTIVE', undefined, { marginBreach: true }),
    'critical'
  );
  assert.equal(resolveCampaignListRowAlert('EXHAUSTED'), 'warning');
  assert.equal(resolveCampaignListRowAlert('ACTIVE', undefined, { budgetUsedPct: 92 }), 'warning');
  assert.equal(resolveCampaignListRowAlert('PAUSED', undefined, { budgetUsedPct: 92 }), 'none');
  assert.equal(resolveCampaignListRowAlert('ACTIVE'), 'none');
});

test('campaignListRowDataAttributes maps accent tones to data-row-accent', () => {
  assert.deepEqual(campaignListRowDataAttributes(false, 'muted'), { 'data-row-accent': 'muted' });
  assert.deepEqual(campaignListRowDataAttributes(false, 'warning'), {
    'data-row-accent': 'warning',
  });
  assert.deepEqual(campaignListRowDataAttributes(false, 'critical'), {
    'data-row-accent': 'critical',
  });
});

test('selected row uses data-row-selected only', () => {
  assert.deepEqual(campaignListRowDataAttributes(true), { 'data-row-selected': true });
});

test('campaignStatusCellClass fills status column with muted tone colors', () => {
  assert.match(campaignStatusCellClass('ACTIVE'), /bg-admin-status-active\/20/);
  assert.match(campaignStatusCellClass('PAUSED'), /bg-admin-status-paused\/20/);
  assert.match(campaignStatusCellClass('EXHAUSTED'), /bg-muted/);
});

test('campaignStatusBadgeClass tints active campaigns green', () => {
  assert.match(campaignStatusBadgeClass('ACTIVE'), /text-admin-status-active/);
  assert.match(campaignStatusBadgeClass('PAUSED'), /text-admin-status-paused/);
  assert.match(campaignStatusBadgeClass('ACTIVE'), /rounded-none/);
});
