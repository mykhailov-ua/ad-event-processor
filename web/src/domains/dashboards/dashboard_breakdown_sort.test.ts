import assert from 'node:assert/strict';
import test from 'node:test';

import type { DashboardBreakdownRow } from '@/domains/dashboards/buyer_dashboard_types';
import {
  compareDashboardBreakdownRows,
  DEFAULT_CAMPAIGNS_BREAKDOWN_SORT,
  sortDashboardBreakdownRows,
  toggleDashboardBreakdownSort,
} from '@/domains/dashboards/dashboard_breakdown_sort';

const row = (id: string, profitMicro: number, name?: string): DashboardBreakdownRow => ({
  id,
  name: name ?? id,
  profit_micro: profitMicro,
});

test('DEFAULT_CAMPAIGNS_BREAKDOWN_SORT sorts profit ascending', () => {
  assert.deepEqual(DEFAULT_CAMPAIGNS_BREAKDOWN_SORT, { column: 'profit', order: 'asc' });
  const sorted = sortDashboardBreakdownRows(
    [row('a', 100), row('b', -50), row('c', 0)],
    DEFAULT_CAMPAIGNS_BREAKDOWN_SORT
  );
  assert.deepEqual(
    sorted.map((item) => item.id),
    ['b', 'c', 'a']
  );
});

test('compareDashboardBreakdownRows uses name tie-break', () => {
  const left = row('z', 10);
  const right = row('a', 10);
  assert.ok(compareDashboardBreakdownRows(left, right, { column: 'profit', order: 'asc' }) > 0);
});

test('toggleDashboardBreakdownSort flips order on same column', () => {
  const next = toggleDashboardBreakdownSort({ column: 'profit', order: 'asc' }, 'profit');
  assert.deepEqual(next, { column: 'profit', order: 'desc' });
});
