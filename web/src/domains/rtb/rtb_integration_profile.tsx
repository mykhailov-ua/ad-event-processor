import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import type { RtbIntegrationProfile } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';

export type RtbIntegrationProfilePanelProps = {
  profile: RtbIntegrationProfile | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
};

export function RtbIntegrationProfilePanel({
  profile,
  fetching,
  error,
  hasSnapshot,
  licenseGated,
}: RtbIntegrationProfilePanelProps) {
  if (licenseGated) {
    return (
      <PageChrome title="RTB integration profile">
        <RtbNav />
        <RtbLicenseStub />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="RTB integration profile">
        <RtbNav />
        {rtbPanelError(error, 'Could not load integration profile')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="RTB integration profile">
      <RtbNav />

      {profile ? (
        <JsonDashboardView payload={profile as Record<string, unknown>} />
      ) : (
        rtbPanelError(new Error('Empty profile'), 'Could not load integration profile')
      )}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
