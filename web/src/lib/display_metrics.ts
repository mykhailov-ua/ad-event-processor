export function formatDashboardRoiPct(value: number): string {
  if (!Number.isFinite(value)) {
    return '';
  }
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}

export function formatDashboardCrPct(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(2)}%`;
}
