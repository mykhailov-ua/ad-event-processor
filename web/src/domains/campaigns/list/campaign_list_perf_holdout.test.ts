import assert from 'node:assert/strict';
import test from 'node:test';

import {
  defaultCampaignListColumnPrefs,
  moveDataColumn,
} from '@/domains/campaigns/list/campaign_list_columns';
import { computeCampaignListColumnWidths } from '@/domains/campaigns/list/campaign_list_column_widths';
import {
  CAMPAIGN_LIST_MOVE_COLUMN_BUDGET,
  CAMPAIGN_LIST_WIDTHS_100_BUDGET,
  PERF_TOLERANCE_RATIO,
} from '@/lib/perf/budgets';
import { buildLargeCampaignListFixture } from '@/lib/perf/fixtures/campaign_list_large';
import { measureMedianMs } from '@/lib/perf/measure';

test('moveDataColumn holdout: 1000 reorders stay within perf budget', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder;
  const medianMs = measureMedianMs(() => {
    let draft = order;
    for (let i = 0; i < 1000; i += 1) {
      draft = moveDataColumn(draft, 'clicks', 'impressions');
      draft = moveDataColumn(draft, 'impressions', 'clicks');
    }
  }, 5);

  const perOpMs = medianMs / 1000;
  const limitMs = CAMPAIGN_LIST_MOVE_COLUMN_BUDGET.medianMs * PERF_TOLERANCE_RATIO;
  assert.ok(
    perOpMs <= limitMs,
    `moveDataColumn per-op ${perOpMs.toFixed(4)} ms exceeds ${limitMs.toFixed(4)} ms`
  );
});

test('column width probe holdout: 100 rows within budget', () => {
  const fixture = buildLargeCampaignListFixture(100);
  const medianMs = measureMedianMs(() => {
    computeCampaignListColumnWidths({
      columns: fixture.columns,
      items: fixture.items,
      metricsById: fixture.metricsById,
      marginsById: {},
      customerNameById: fixture.customerNameById,
    });
  }, 5);

  const limitMs = CAMPAIGN_LIST_WIDTHS_100_BUDGET.medianMs * PERF_TOLERANCE_RATIO;
  assert.ok(
    medianMs <= limitMs,
    `computeCampaignListColumnWidths ${medianMs.toFixed(3)} ms exceeds ${limitMs.toFixed(3)} ms`
  );
});

test('column width probe holdout: doubling rows is sub-quadratic', () => {
  const small = buildLargeCampaignListFixture(50);
  const large = buildLargeCampaignListFixture(100);

  const time50 = measureMedianMs(() => {
    computeCampaignListColumnWidths({
      columns: small.columns,
      items: small.items,
      metricsById: small.metricsById,
      marginsById: {},
      customerNameById: small.customerNameById,
    });
  }, 5);

  const time100 = measureMedianMs(() => {
    computeCampaignListColumnWidths({
      columns: large.columns,
      items: large.items,
      metricsById: large.metricsById,
      marginsById: {},
      customerNameById: large.customerNameById,
    });
  }, 5);

  const ratio = time100 / Math.max(time50, 0.001);
  assert.ok(
    ratio < 2.5,
    `width probe scaled ${ratio.toFixed(2)}x for 2x rows (expected < 2.5x linear-ish)`
  );
});
