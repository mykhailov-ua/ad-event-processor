import { SectionNav } from '@/shell/section_nav';
import { panelError } from '@/shell/panel_error';
import type { SectionNavItem } from '@/lib/nav_config';

export const CREATIVE_NAV_ITEMS: SectionNavItem[] = [
  { path: '/creative', label: 'Hub', exact: true },
  { path: '/flows', label: 'Flows' },
  { path: '/landers', label: 'Landers' },
  { path: '/offers', label: 'Offers' },
  { path: '/brands', label: 'Brands' },
  { path: '/supply', label: 'Supply' },
  { path: '/domains', label: 'Domains' },
];

export function CreativeNav() {
  return <SectionNav items={CREATIVE_NAV_ITEMS} label="Creative sections" />;
}

export function creativePanelError(error: Error, title: string) {
  return panelError(error, title);
}
