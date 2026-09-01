/** Display-only spend/budget ratio from wire decimal strings (0..1). */
export function budgetUtilizationRatio(
  spend?: string | null,
  budget?: string | null,
): number | undefined {
  const spendAmount = Number.parseFloat(spend?.trim() ?? '');
  const budgetAmount = Number.parseFloat(budget?.trim() ?? '');
  if (!Number.isFinite(spendAmount) || !Number.isFinite(budgetAmount) || budgetAmount <= 0) {
    return undefined;
  }
  return Math.min(1, Math.max(0, spendAmount / budgetAmount));
}

export function budgetUtilizationPercent(
  spend?: string | null,
  budget?: string | null,
): number | undefined {
  const ratio = budgetUtilizationRatio(spend, budget);
  if (ratio == null) {
    return undefined;
  }
  return Math.round(ratio * 1000) / 10;
}
