import { SectionNav } from '@/shell/section_nav';
import type { SectionNavItem } from '@/lib/nav_config';

const DASHBOARD_NAV_ITEMS: SectionNavItem[] = [
  { path: '/dashboards/buyer', label: 'Buyer' },
  { path: '/dashboards/adops', label: 'Ad ops' },
];

export function DashboardNav() {
  return <SectionNav items={DASHBOARD_NAV_ITEMS} label="Dashboard views" />;
}
