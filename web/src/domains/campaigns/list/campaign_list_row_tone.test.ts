import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignListRowClass,
  campaignStatusBadgeClass,
  campaignStatusCellClass,
  isInactiveCampaignStatus,
  normalizeCampaignStatus,
  resolveCampaignListRowAccent,
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
  assert.ok(campaignListRowClass(false).includes('odd:[&_td]:bg-card'));
  assert.ok(campaignListRowClass(false).includes('[&_td[data-col-pin]]:bg-admin-table-pin'));
  assert.equal(campaignListRowClass(false).includes('[&>td]:!bg-primary/22'), false);
});

test('campaignListRowClass_holdout: pinned cells use opaque pin background', () => {
  const rowClass = campaignListRowClass(false);
  assert.equal(rowClass.includes('bg-admin-table-pin/90'), false);
  assert.ok(rowClass.includes('even:[&_td[data-col-pin]]:bg-admin-table-pin'));
});

test('resolveCampaignListRowAccent maps lifecycle and problem signals', () => {
  assert.equal(resolveCampaignListRowAccent('PAUSED'), 'muted');
  assert.equal(resolveCampaignListRowAccent('ARCHIVED'), 'muted');
  assert.equal(
    resolveCampaignListRowAccent('ACTIVE', undefined, { marginBreach: true }),
    'critical'
  );
  assert.equal(resolveCampaignListRowAccent('EXHAUSTED'), 'warning');
  assert.equal(
    resolveCampaignListRowAccent('ACTIVE', undefined, { budgetUsedPct: 92 }),
    'warning'
  );
  assert.equal(resolveCampaignListRowAccent('ACTIVE'), 'none');
});

test('campaignListRowClass tints whole row for muted warning and critical accents', () => {
  assert.ok(campaignListRowClass(false, 'muted').includes('[&>td:not([data-col-pin])]:!bg-muted/30'));
  assert.ok(
    campaignListRowClass(false, 'warning').includes('[&>td:not([data-col-pin])]:!bg-admin-warn-bg/80')
  );
  assert.ok(
    campaignListRowClass(false, 'critical').includes('[&>td:not([data-col-pin])]:!bg-destructive/12')
  );
});

test('selected row tints scrollable cells and left rail on first column', () => {
  assert.ok(campaignListRowClass(true).includes('[&>td:not([data-col-pin])]:!bg-primary/22'));
  assert.ok(
    campaignListRowClass(true).includes('[&>td:first-child]:shadow-[inset_3px_0_0_0_hsl(var(--primary))]')
  );
  assert.equal(campaignListRowClass(true).includes('[&>td]:!bg-primary/22'), false);
  assert.equal(campaignListRowClass(true).includes('shadow-[inset_3px_0_0_0_hsl(var(--primary))]'), false);
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
