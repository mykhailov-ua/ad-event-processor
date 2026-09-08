import type { ReactNode } from 'react';

import type { DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import type { DashboardBreakdownEntityId } from '@/domains/dashboards/dashboard_preferences';

export type DashboardBreakdownSectionConfig = {
  id: DashboardBreakdownEntityId;
  title: string;
  table: DashboardBreakdownTable | undefined;
  nameLink?: (row: { id?: string; name?: string }) => ReactNode;
};

export const DASHBOARD_BREAKDOWN_LAYOUT_ORDER: readonly DashboardBreakdownEntityId[] = [
  'campaigns',
  'landers',
  'offers',
  'sources',
];

export type DashboardTableGridSlot =
  | { kind: 'breakdown'; section: DashboardBreakdownSectionConfig }
  | { kind: 'recent_clicks' };

export type DashboardTablesLayout = {
  top: DashboardBreakdownSectionConfig | null;
  pairRows: DashboardTableGridSlot[][];
};

export function buildDashboardTablesLayout(
  sections: readonly DashboardBreakdownSectionConfig[],
  includeRecentClicks: boolean
): DashboardTablesLayout {
  const byId = new Map(sections.map((section) => [section.id, section]));
  const ordered = DASHBOARD_BREAKDOWN_LAYOUT_ORDER.flatMap((id) => {
    const section = byId.get(id);
    return section ? [section] : [];
  });

  const top = ordered[0] ?? null;
  const restSlots: DashboardTableGridSlot[] = ordered.slice(1).map((section) => ({
    kind: 'breakdown',
    section,
  }));
  if (includeRecentClicks) {
    restSlots.push({ kind: 'recent_clicks' });
  }

  const pairRows: DashboardTableGridSlot[][] = [];
  for (let index = 0; index < restSlots.length; index += 2) {
    pairRows.push(restSlots.slice(index, index + 2));
  }

  return { top, pairRows };
}
