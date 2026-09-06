import type { NavItem } from '@/lib/nav_config';
import { filterNavItems } from '@/lib/nav_config';
import { NAV_FILTER_1K_BUDGET } from '@/lib/perf/budgets';
import {
  assertPerfBudget,
  formatBenchResult,
  runPerfBudget,
  type BenchResult,
} from '@/lib/perf/measure';

function buildSyntheticNavCatalog(size: number): NavItem[] {
  const items: NavItem[] = [];
  for (let i = 0; i < size; i += 1) {
    items.push({
      path: `/reports/report-${i}`,
      label: `Report ${i} performance funnel`,
      permission: i % 2 === 0 ? 'campaigns:read' : 'audit:read',
    });
  }
  return items;
}

export function benchNavFilterHotPath(): BenchResult[] {
  const catalog = buildSyntheticNavCatalog(1000);
  const permissions = ['campaigns:read', 'audit:read', 'settings:read'];

  const result = runPerfBudget(NAV_FILTER_1K_BUDGET, () => {
    for (let i = 0; i < 1000; i += 1) {
      filterNavItems(catalog, permissions);
    }
  });

  return [result];
}

export function runNavFilterHotPathBench(): void {
  const results = benchNavFilterHotPath();
  for (const result of results) {
    console.log(`bench ok: ${formatBenchResult(result)}`);
    assertPerfBudget(result);
  }
}
