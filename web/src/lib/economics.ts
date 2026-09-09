export type EconomicsMicroRow = {
  revenue_micro?: number;
  cost_micro?: number;
  spend_micro?: number;
  ad_spend_micro?: number;
  profit_micro?: number;
  roi_pct?: number;
};

/** Wire cost field preference only; no derived totals. */
export function resolveEconomicsCostMicro(row: EconomicsMicroRow): number | undefined {
  if (row.cost_micro != null && Number.isFinite(row.cost_micro)) {
    return row.cost_micro;
  }
  if (row.spend_micro != null && Number.isFinite(row.spend_micro)) {
    return row.spend_micro;
  }
  if (row.ad_spend_micro != null && Number.isFinite(row.ad_spend_micro)) {
    return row.ad_spend_micro;
  }
  return undefined;
}

/** Server-authoritative profit micro; optional explicit fallback (e.g. true_profit_micro). */
export function resolveEconomicsProfitMicro(
  row: EconomicsMicroRow,
  profitMicroFallback?: number
): number | undefined {
  if (row.profit_micro != null && Number.isFinite(row.profit_micro)) {
    return row.profit_micro;
  }
  if (profitMicroFallback != null && Number.isFinite(profitMicroFallback)) {
    return profitMicroFallback;
  }
  return undefined;
}

/** Server-authoritative ROI percent; optional explicit fallback (e.g. true_roi_pct). */
export function resolveEconomicsRoiPct(
  row: EconomicsMicroRow,
  roiPctFallback?: number
): number | undefined {
  if (row.roi_pct != null && Number.isFinite(row.roi_pct)) {
    return row.roi_pct;
  }
  if (roiPctFallback != null && Number.isFinite(roiPctFallback)) {
    return roiPctFallback;
  }
  return undefined;
}
