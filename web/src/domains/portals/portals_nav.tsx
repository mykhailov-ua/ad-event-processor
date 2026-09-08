import { SectionNav } from '@/shell/section_nav';
import { panelError } from '@/shell/panel_error';
import type { SectionNavItem } from '@/lib/nav_config';

export const PORTALS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/portals', label: 'Hub', exact: true },
  { path: '/selfserve', label: 'Self-serve' },
  { path: '/publisher/dashboard', label: 'Publisher' },
  { path: '/telegram', label: 'Telegram' },
  { path: '/report-schedules', label: 'Report schedules' },
  { path: '/views', label: 'Saved views' },
  { path: '/forecast/campaign', label: 'Forecast' },
];

export function PortalsNav() {
  return <SectionNav items={PORTALS_NAV_ITEMS} label="Portals sections" />;
}

export function portalsPanelError(error: Error, title: string) {
  return panelError(error, title);
}
