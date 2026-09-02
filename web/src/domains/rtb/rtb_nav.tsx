import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { SectionNav } from '@/shell/section_nav';
import { StubBanner } from '@/shell/stub_banner';
import type { SectionNavItem } from '@/lib/nav_config';

export const RTB_NAV_ITEMS: SectionNavItem[] = [
  { path: '/rtb', label: 'Reports', exact: true },
  { path: '/rtb/deals', label: 'Deals' },
  { path: '/rtb/shadow', label: 'Shadow' },
  { path: '/rtb/floors', label: 'Floors' },
  { path: '/rtb/validate', label: 'Validate' },
  { path: '/rtb/integration-profile', label: 'Integration' },
];

export function RtbNav() {
  return <SectionNav items={RTB_NAV_ITEMS} label="RTB sections" />;
}

export function rtbLicenseGated(error: Error | undefined): boolean {
  return error instanceof ApiError && error.status === 403;
}

export function RtbLicenseStub() {
  return (
    <StubBanner
      title="OpenRTB license required"
      message="RTB admin requires the openrtb license feature. Contact your operator to enable RTB."
    />
  );
}

export function rtbPanelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 403) {
    return <RtbLicenseStub />;
  }
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}
