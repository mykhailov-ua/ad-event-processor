import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignBudgetUsedPercent,
  formatBudgetUsedPercent,
} from './campaign_budget_used.ts';

test('campaignBudgetUsedPercent returns share of budget spent', () => {
  assert.equal(campaignBudgetUsedPercent('100.00', '25.50'), 25.5);
  assert.equal(campaignBudgetUsedPercent('100.00', '150.00'), 100);
});

test('campaignBudgetUsedPercent returns undefined for invalid budget', () => {
  assert.equal(campaignBudgetUsedPercent('', '10'), undefined);
  assert.equal(campaignBudgetUsedPercent('0', '10'), undefined);
  assert.equal(campaignBudgetUsedPercent('abc', '10'), undefined);
});

test('formatBudgetUsedPercent formats small and capped values', () => {
  assert.equal(formatBudgetUsedPercent(0.05), '<0.1%');
  assert.equal(formatBudgetUsedPercent(100), '100%');
  assert.equal(formatBudgetUsedPercent(33.333), '33.3%');
});
