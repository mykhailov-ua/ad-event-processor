import { computeCampaignListColumnWidths } from '@/domains/campaigns/list/campaign_list_column_widths';
import { buildCampaignRowVm } from '@/domains/campaigns/list/campaign_list_row_vm';
import {
  defaultCampaignListColumnPrefs,
  moveDataColumn,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  CAMPAIGN_LIST_MOVE_COLUMN_BUDGET,
  CAMPAIGN_LIST_ROW_VM_100_BUDGET,
  CAMPAIGN_LIST_WIDTHS_100_BUDGET,
} from '@/lib/perf/budgets';
import { buildLargeCampaignListFixture } from '@/lib/perf/fixtures/campaign_list_large';
import {
  assertPerfBudget,
  formatBenchResult,
  runPerfBudget,
  type BenchResult,
} from '@/lib/perf/measure';

export function benchCampaignListHotPath(): BenchResult[] {
  const fixture100 = buildLargeCampaignListFixture(100);
  const results: BenchResult[] = [];

  results.push(
    runPerfBudget(CAMPAIGN_LIST_WIDTHS_100_BUDGET, () => {
      computeCampaignListColumnWidths({
        columns: fixture100.columns,
        items: fixture100.items,
        metricsById: fixture100.metricsById,
        marginsById: {},
        customerNameById: fixture100.customerNameById,
      });
    })
  );

  results.push(
    runPerfBudget(CAMPAIGN_LIST_ROW_VM_100_BUDGET, () => {
      for (const campaign of fixture100.items) {
        buildCampaignRowVm(
          campaign,
          fixture100.metricsById[campaign.id],
          undefined,
          fixture100.customerNameById,
          {}
        );
      }
    })
  );

  const prefs = defaultCampaignListColumnPrefs();
  const order = prefs.dataColumnOrder;
  results.push(
    runPerfBudget(CAMPAIGN_LIST_MOVE_COLUMN_BUDGET, () => {
      let draft = order;
      for (let i = 0; i < 1000; i += 1) {
        draft = moveDataColumn(draft, 'clicks', 'impressions');
        draft = moveDataColumn(draft, 'impressions', 'clicks');
      }
    })
  );

  return results;
}

export function runCampaignListHotPathBench(): void {
  const results = benchCampaignListHotPath();
  for (const result of results) {
    console.log(`bench ok: ${formatBenchResult(result)}`);
    assertPerfBudget(result);
  }
}
