import {
  buildCampaignListExportCsv,
  buildCampaignListExportRows,
} from '@/domains/campaigns/list/campaign_list_export_rows';
import { defaultCampaignListExportDataColumns } from '@/domains/campaigns/list/campaign_list_columns';
import { CAMPAIGN_LIST_EXPORT_CSV_100_BUDGET } from '@/lib/perf/budgets';
import { buildLargeCampaignListFixture } from '@/lib/perf/fixtures/campaign_list_large';
import {
  assertPerfBudget,
  formatBenchResult,
  runPerfBudget,
  type BenchResult,
} from '@/lib/perf/measure';

export function benchCampaignListExportColdPath(): BenchResult[] {
  const fixture100 = buildLargeCampaignListFixture(100);
  const columns = defaultCampaignListExportDataColumns();
  const rows = buildCampaignListExportRows(
    fixture100.items,
    columns,
    fixture100.metricsById,
    {},
    fixture100.customerNameById,
    {}
  );

  const result = runPerfBudget(CAMPAIGN_LIST_EXPORT_CSV_100_BUDGET, () => {
    buildCampaignListExportCsv(columns, rows);
  });

  return [result];
}

export function runCampaignListExportColdPathBench(): void {
  const results = benchCampaignListExportColdPath();
  for (const result of results) {
    console.log(`bench ok: ${formatBenchResult(result)}`);
    assertPerfBudget(result);
  }
}
