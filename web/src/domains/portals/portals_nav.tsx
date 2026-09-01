import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/components/system/error_block';
import { SectionNav } from '@/components/system/section_nav';
import { StubBanner } from '@/components/system/stub_banner';
import type { SectionNavItem } from '@/lib/nav_config';

export const PORTALS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/portals', label: 'Hub', exact: true },
  { path: '/selfserve', label: 'Self-serve' },
  { path: '/publisher/dashboard', label: 'Publisher' },
  { path: '/telegram', label: 'Telegram' },
  { path: '/report-schedules', label: 'Report schedules' },
  { path: '/views', label: 'Saved views' },
  { path: '/forecast/campaign', label: 'Forecast' },
];

export function PortalsNav() {
  return <SectionNav items={PORTALS_NAV_ITEMS} label="Portals sections" />;
}

export function portalsPanelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 403) {
    return <StubBanner title={`${title} forbidden`} message={error.message} />;
  }
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}
