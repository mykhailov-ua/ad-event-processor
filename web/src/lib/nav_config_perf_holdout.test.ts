import assert from 'node:assert/strict';
import test from 'node:test';

import type { NavItem } from '@/lib/nav_config';
import { filterNavItems } from '@/lib/nav_config';
import { NAV_FILTER_1K_BUDGET, PERF_TOLERANCE_RATIO } from '@/lib/perf/budgets';
import { measureMedianMs } from '@/lib/perf/measure';

function buildSyntheticNavCatalog(size: number): NavItem[] {
  const items: NavItem[] = [];
  for (let i = 0; i < size; i += 1) {
    items.push({
      path: `/section/item-${i}`,
      label: `Nav item ${i}`,
      permission: i % 2 === 0 ? 'campaigns:read' : 'audit:read',
    });
  }
  return items;
}

test('filterNavItems holdout: 1k catalog x 1k filters within budget', () => {
  const catalog = buildSyntheticNavCatalog(1000);
  const permissions = ['campaigns:read', 'audit:read'];

  const medianMs = measureMedianMs(() => {
    for (let i = 0; i < 1000; i += 1) {
      filterNavItems(catalog, permissions);
    }
  }, 5);

  const perFilterMs = medianMs / 1000;
  const limitMs = NAV_FILTER_1K_BUDGET.medianMs * PERF_TOLERANCE_RATIO;
  assert.ok(
    perFilterMs <= limitMs,
    `filterNavItems per-call ${perFilterMs.toFixed(4)} ms exceeds ${limitMs.toFixed(4)} ms`,
  );
});
