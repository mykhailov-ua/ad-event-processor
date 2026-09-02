import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { SectionNav } from '@/shell/section_nav';
import { StubBanner } from '@/shell/stub_banner';
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
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  if (error instanceof ApiError && error.status === 403) {
    return <StubBanner title={`${title} forbidden`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}
