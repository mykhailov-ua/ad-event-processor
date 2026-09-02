import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { SectionNav } from '@/shell/section_nav';
import { StubBanner } from '@/shell/stub_banner';
import type { SectionNavItem } from '@/lib/nav_config';

export const OPS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/ops', label: 'Home', exact: true },
  { path: '/ops/dlq', label: 'DLQ inbox' },
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
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock error={error} title={title} />;
}
