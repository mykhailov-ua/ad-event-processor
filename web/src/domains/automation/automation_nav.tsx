import { SectionNav } from '@/shell/section_nav';
import { panelError } from '@/shell/panel_error';
import type { SectionNavItem } from '@/lib/nav_config';

export const AUTOMATION_NAV_ITEMS: SectionNavItem[] = [
  { path: '/automation', label: 'Overview', exact: true },
  { path: '/automation/presets', label: 'Catalog' },
  { path: '/automation/rules', label: 'Rules' },
  { path: '/traffic-optimizer/rules', label: 'Optimizer' },
  { path: '/smart-alerts/rules', label: 'Alerts' },
  { path: '/smart-alerts/history', label: 'History' },
  { path: '/margin-guard/policies', label: 'Margin' },
];

export function AutomationNav() {
  return <SectionNav items={AUTOMATION_NAV_ITEMS} label="Automation sections" />;
}

export function automationPanelError(error: Error, title: string) {
  return panelError(error, title);
}
