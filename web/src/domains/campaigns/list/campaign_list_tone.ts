import type { CampaignMargin } from '@/api/types';
import {
  adminMetricMutedZeroClass,
  adminMetricNegativeClass,
  adminMetricPositiveClass,
} from '@/lib/admin_metric_tone';
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
    return adminMetricMutedZeroClass;
  }
  return profitMicro > 0 ? adminMetricPositiveClass : adminMetricNegativeClass;
}

export function roiToneClassFromRate(roi: VmRateCell): string {
  if (roi.isZero || roi.text === '-') {
    return adminMetricMutedZeroClass;
  }
  return roi.valPct >= 0 ? adminMetricPositiveClass : adminMetricNegativeClass;
}
