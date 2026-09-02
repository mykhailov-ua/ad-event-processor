import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { SectionNav } from '@/shell/section_nav';
import { StubBanner } from '@/shell/stub_banner';
import type { SectionNavItem } from '@/lib/nav_config';

export const INTEGRATIONS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/integrations', label: 'Hub', exact: true },
  { path: '/integrations/cost-sync', label: 'Cost sync' },
  { path: '/integrations/postbacks', label: 'Postbacks' },
  { path: '/integrations/schemas', label: 'Schemas' },
  { path: '/integrations/platform-campaigns', label: 'Platform links' },
  { path: '/integrations/affiliate-presets', label: 'Affiliate presets' },
];

export function IntegrationsNav() {
  return <SectionNav items={INTEGRATIONS_NAV_ITEMS} label="Integrations sections" />;
}

export function integrationsPanelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}
