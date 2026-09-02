import { SectionNav } from '@/shell/section_nav';
import type { SectionNavItem } from '@/lib/nav_config';

export const SETTINGS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/settings', label: 'Platform', exact: true },
  { path: '/settings/license', label: 'License' },
];

export function SettingsNav() {
  return <SectionNav items={SETTINGS_NAV_ITEMS} label="Settings sections" />;
}
