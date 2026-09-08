import assert from 'node:assert/strict';
import test from 'node:test';

import { probeDashboardBreakdownDataColumnWidthPx } from './dashboard_table_column_widths.ts';

test('probeDashboardBreakdownDataColumnWidthPx fits roi and profit labels', () => {
  const rows = [
    {
      name: 'Offer A',
      profit_micro: 14_892_000_000,
      roi_pct: 531.89,
    },
  ];
  const totals = {
    name: 'Total',
    profit_micro: 73_100_000_000,
    roi_pct: 502.14,
  };

  assert.ok(probeDashboardBreakdownDataColumnWidthPx('profit', rows, totals) >= 100);
  assert.ok(probeDashboardBreakdownDataColumnWidthPx('roi', rows, totals) >= 104);
  assert.ok(probeDashboardBreakdownDataColumnWidthPx('roi', rows, totals) > 56);
});
