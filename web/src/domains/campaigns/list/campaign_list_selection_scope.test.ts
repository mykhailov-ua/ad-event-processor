import assert from 'node:assert/strict';
import test from 'node:test';

import { campaignListSelectionScopeKey } from './campaign_list_selection_scope.ts';

test('campaignListSelectionScopeKey changes when pagination offset changes', () => {
  const base = {
    query: { limit: 50, offset: 0, sort: 'name', order: 'asc' as const },
    statsFrom: '2026-01-01T00:00:00.000Z',
    statsTo: '2026-01-08T00:00:00.000Z',
  };
  const pageTwo = {
    ...base,
    query: { ...base.query, offset: 50 },
  };

  assert.notEqual(campaignListSelectionScopeKey(base), campaignListSelectionScopeKey(pageTwo));
});

test('campaignListSelectionScopeKey_holdout ignores refresh-only churn', () => {
  const a = {
    query: { limit: 50, offset: 0, status: 'ACTIVE', sort: 'name', order: 'asc' as const },
  };
  const b = {
    query: { limit: 50, offset: 0, status: 'ACTIVE', sort: 'name', order: 'asc' as const },
  };

  assert.equal(campaignListSelectionScopeKey(a), campaignListSelectionScopeKey(b));
});
