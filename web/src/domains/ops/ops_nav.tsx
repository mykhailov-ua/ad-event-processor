import { SectionNav } from '@/shell/section_nav';
import { panelError } from '@/shell/panel_error';
import type { SectionNavItem } from '@/lib/nav_config';

export const OPS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/ops', label: 'Home', exact: true },
  { path: '/ops/health', label: 'Health' },
  { path: '/ops/sync-errors', label: 'Sync errors' },
  { path: '/ops/blacklist', label: 'Blacklist' },
  { path: '/ops/incidents', label: 'Incidents' },
  { path: '/ops/outbox', label: 'Outbox' },
  { path: '/ops/shards', label: 'Shards' },
  { path: '/ops/ml-model', label: 'ML model' },
  { path: '/ops/domains', label: 'Domains' },
  { path: '/ops/recon', label: 'Recon' },
  { path: '/ops/consent', label: 'Consent proofs' },
  { path: '/ops/rum', label: 'RUM' },
  { path: '/ops/metrics', label: 'Metrics' },
];

export function OpsNav({ variant = 'admin' }: { variant?: 'pill' | 'admin' }) {
  return <SectionNav items={OPS_NAV_ITEMS} label="Ops sections" variant={variant} />;
}

export function opsPanelError(error: Error, title: string) {
  return panelError(error, title);
}
