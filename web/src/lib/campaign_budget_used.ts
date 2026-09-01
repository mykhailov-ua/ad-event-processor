export function campaignBudgetUsedPercent(
  budgetLimit?: string | null,
  currentSpend?: string | null,
): number | undefined {
  const budget = Number.parseFloat(budgetLimit ?? '');
  const spend = Number.parseFloat(currentSpend ?? '');
  if (!Number.isFinite(budget) || budget <= 0 || !Number.isFinite(spend)) {
    return undefined;
  }
  return Math.min(100, Math.max(0, (spend / budget) * 100));
}

export function formatBudgetUsedPercent(percent: number): string {
  if (percent >= 100) {
    return '100%';
  }
  if (percent < 0.1 && percent > 0) {
    return '<0.1%';
  }
  return `${percent.toFixed(1)}%`;
}
