export type DashboardRangePreset =
  | 'today'
  | 'yesterday'
  | '7d'
  | '30d'
  | 'this_month'
  | 'custom';

export function dashboardPresetRange(preset: DashboardRangePreset): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to);

  switch (preset) {
    case 'today': {
      from.setUTCHours(0, 0, 0, 0);
      return { from: from.toISOString(), to: to.toISOString() };
    }
    case 'yesterday': {
      to.setUTCHours(0, 0, 0, 0);
      from.setUTCDate(from.getUTCDate() - 1);
      from.setUTCHours(0, 0, 0, 0);
      return { from: from.toISOString(), to: to.toISOString() };
    }
    case '30d': {
      from.setUTCDate(from.getUTCDate() - 30);
      return { from: from.toISOString(), to: to.toISOString() };
    }
    case 'this_month': {
      from.setUTCDate(1);
      from.setUTCHours(0, 0, 0, 0);
      return { from: from.toISOString(), to: to.toISOString() };
    }
    case '7d':
    default: {
      from.setUTCDate(from.getUTCDate() - 7);
      return { from: from.toISOString(), to: to.toISOString() };
    }
  }
}

export const DASHBOARD_RANGE_PRESETS: Array<{ id: DashboardRangePreset; label: string }> = [
  { id: 'today', label: 'Today' },
  { id: 'yesterday', label: 'Yesterday' },
  { id: '7d', label: 'Last 7 days' },
  { id: '30d', label: 'Last 30 days' },
  { id: 'this_month', label: 'This month' },
  { id: 'custom', label: 'Custom' },
];
