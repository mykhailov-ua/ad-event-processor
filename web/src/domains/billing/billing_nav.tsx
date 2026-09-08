import { SectionNav } from '@/shell/section_nav';
import { panelError } from '@/shell/panel_error';
import type { SectionNavItem } from '@/lib/nav_config';

export const BILLING_NAV_ITEMS: SectionNavItem[] = [
  { path: '/billing', label: 'Overview', exact: true },
  { path: '/billing/exports', label: 'Ledger exports' },
];

export function BillingNav() {
  return <SectionNav items={BILLING_NAV_ITEMS} label="Billing sections" />;
}

export function billingPanelError(error: Error, title: string) {
  return panelError(error, title);
}
