export type EconomicsMicroRow = {
  revenue_micro?: number;
  cost_micro?: number;
  spend_micro?: number;
  ad_spend_micro?: number;
  profit_micro?: number;
  roi_pct?: number;
};

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

export function resolveEconomicsProfitMicro(
  row: EconomicsMicroRow,
  profitMicroFallback?: number
): number | undefined {
  const revenueMicro = row.revenue_micro;
  const costMicro = resolveEconomicsCostMicro(row);
  if (
    revenueMicro != null &&
    costMicro != null &&
    Number.isFinite(revenueMicro) &&
    Number.isFinite(costMicro)
  ) {
    return revenueMicro - costMicro;
  }
  if (profitMicroFallback != null && Number.isFinite(profitMicroFallback)) {
    return profitMicroFallback;
  }
  return row.profit_micro;
}

export function resolveEconomicsRoiPct(
  row: EconomicsMicroRow,
  roiPctFallback?: number
): number | undefined {
  const costMicro = resolveEconomicsCostMicro(row);
  const profitMicro = resolveEconomicsProfitMicro(row);
  if (costMicro != null && costMicro > 0 && profitMicro != null) {
    return (profitMicro / costMicro) * 100;
  }
  if (roiPctFallback != null && Number.isFinite(roiPctFallback)) {
    return roiPctFallback;
  }
  return row.roi_pct;
}
