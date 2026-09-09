import { buildCampaignRowVm } from '@/domains/campaigns/list/campaign_list_row_vm';
import { CAMPAIGN_LIST_ROW_VM_100_BUDGET } from '@/lib/perf/budgets';
import { buildLargeCampaignListFixture } from '@/lib/perf/fixtures/campaign_list_large';
import {
  assertPerfBudget,
  formatBenchResult,
  runPerfBudget,
  type BenchResult,
} from '@/lib/perf/measure';

export function benchCampaignListHotPath(): BenchResult[] {
  const fixture100 = buildLargeCampaignListFixture(100);

  return [
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
    }),
  ];
}

export function runCampaignListHotPathBench(): void {
  const results = benchCampaignListHotPath();
  for (const result of results) {
    console.log(`bench ok: ${formatBenchResult(result)}`);
    assertPerfBudget(result);
  }
}
