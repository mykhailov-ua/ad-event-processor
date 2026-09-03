import type { CampaignMargin } from '@/api/types';
import { formatTableRoi } from '@/domains/campaigns/list/campaign_list_format';

// Re-export row/status tone helpers so callers import from one place.
export {
  campaignListRowClass,
  campaignListRowStatusEdgeClass,
  campaignStatusBadgeClass,
  resolvePerformanceRowTone,
} from '@/domains/campaigns/list/campaign_list_row_tone';

export {
  rateBenchmarkToneClass,
  percentRate,
} from '@/domains/campaigns/list/campaign_list_rate_tone';

/**
 * CSS tone class for the profit cell based on operator_margin_micro.
 * Returns 'font-semibold text-green-700 dark:text-green-400', 'font-semibold text-red-700 dark:text-red-400', or 'tabular-nums text-zinc-400 dark:text-zinc-600'.
 */
export function profitToneClass(margin?: CampaignMargin): string {
  const profitMicro = margin?.operator_margin_micro;
  if (profitMicro == null || profitMicro === 0) {
    return 'tabular-nums text-zinc-400 dark:text-zinc-600';
  }
  return profitMicro > 0 ? 'font-semibold text-green-700 dark:text-green-400' : 'font-semibold text-red-700 dark:text-red-400';
}

/**
 * CSS tone class for the ROI cell.
 * Returns 'font-semibold text-green-700 dark:text-green-400', 'font-semibold text-red-700 dark:text-red-400', or 'tabular-nums text-zinc-400 dark:text-zinc-600'.
 */
export function roiToneClass(margin?: CampaignMargin): string {
  const roi = formatTableRoi(margin?.operator_margin_micro, margin?.rtb_cost_micro);
  if (roi.isZero) {
    return 'tabular-nums text-zinc-400 dark:text-zinc-600';
  }
  return roi.valPct >= 0 ? 'font-semibold text-green-700 dark:text-green-400' : 'font-semibold text-red-700 dark:text-red-400';
}

/**
 * Profit indicator tone for the left-edge column.
 * Distinct from profitToneClass: returns a semantic value, not a CSS class.
 */
export function resolveIndicatorTone(
  margin?: CampaignMargin,
): 'positive' | 'negative' | 'neutral' {
  const profitMicro = margin?.operator_margin_micro;
  if (profitMicro == null || profitMicro === 0) {
    return 'neutral';
  }
  if (profitMicro > 0) {
    return 'positive';
  }
  const roi = formatTableRoi(profitMicro, margin?.rtb_cost_micro);
  if (roi.valPct <= -100) {
    return 'negative';
  }
  return 'negative';
}
