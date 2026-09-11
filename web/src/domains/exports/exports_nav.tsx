import { SectionNav } from '@/shell/section_nav';
import type { SectionNavItem } from '@/lib/nav_config';

export const EXPORTS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/exports', label: 'Hub', exact: true },
  { path: '/exports/schedules', label: 'Schedules' },
];

export function ExportsNav() {
  return <SectionNav items={EXPORTS_NAV_ITEMS} label="Exports sections" />;
}
