import { parseCampaignListColumnPrefs } from '@/domains/campaigns/list/campaign_list_columns';
import { JSON_PARSE_LARGE_PREFS_BUDGET } from '@/lib/perf/budgets';
import { buildLargeCampaignColumnPrefsJson } from '@/lib/perf/fixtures/large_json';
import { assertPerfBudget, formatBenchResult, runPerfBudget, type BenchResult } from '@/lib/perf/measure';

export function benchJsonParseColdPath(): BenchResult[] {
  const raw = buildLargeCampaignColumnPrefsJson();

  const result = runPerfBudget(JSON_PARSE_LARGE_PREFS_BUDGET, () => {
    parseCampaignListColumnPrefs(raw);
  });

  return [result];
}

export function runJsonParseColdPathBench(): void {
  const results = benchJsonParseColdPath();
  for (const result of results) {
    console.log(`bench ok: ${formatBenchResult(result)}`);
    assertPerfBudget(result);
  }
}
