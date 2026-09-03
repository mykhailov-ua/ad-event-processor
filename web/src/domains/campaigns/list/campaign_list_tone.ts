import type { CampaignMargin } from '@/api/types';
import type { VmRateCell } from '@/domains/campaigns/list/campaign_list_row_vm';

export {
  campaignListRowClass,
  campaignStatusBadgeClass,
} from '@/domains/campaigns/list/campaign_list_row_tone';

export {
  rateBenchmarkToneClass,
  percentRate,
} from '@/domains/campaigns/list/campaign_list_rate_tone';

export function profitToneClass(margin?: CampaignMargin): string {
  return profitToneClassFromMicro(margin?.operator_margin_micro);
}

export function profitToneClassFromMicro(profitMicro?: number | null): string {
  if (profitMicro == null || profitMicro === 0) {
    return 'tabular-nums text-zinc-400 dark:text-zinc-600';
  }
  return profitMicro > 0 ? 'font-semibold text-green-700 dark:text-green-400' : 'font-semibold text-red-700 dark:text-red-400';
}

export function roiToneClassFromRate(roi: VmRateCell): string {
  if (roi.isZero || roi.text === '-') {
    return 'tabular-nums text-zinc-400 dark:text-zinc-600';
  }
  return roi.valPct >= 0 ? 'font-semibold text-green-700 dark:text-green-400' : 'font-semibold text-red-700 dark:text-red-400';
}
